import 'dart:async';
import 'dart:typed_data';
import 'package:audioplayers/audioplayers.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:record/record.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:klyra/core/network/dio_client.dart';
import 'package:klyra/features/tutor/data/gemini_live_service.dart';

part 'tutor_session_controller.g.dart';

/// Holds the complete state of the tutor session UI.
class TutorSessionState {
  final SessionState sessionState;
  final String transcript;
  final String? error;
  final bool isMicrophoneActive;

  const TutorSessionState({
    this.sessionState = SessionState.idle,
    this.transcript = '',
    this.error,
    this.isMicrophoneActive = false,
  });

  TutorSessionState copyWith({
    SessionState? sessionState,
    String? transcript,
    String? error,
    bool? isMicrophoneActive,
  }) {
    return TutorSessionState(
      sessionState: sessionState ?? this.sessionState,
      transcript: transcript ?? this.transcript,
      error: error,
      isMicrophoneActive: isMicrophoneActive ?? this.isMicrophoneActive,
    );
  }
}

/// TutorSessionController manages the full lifecycle of a Klyra tutoring session:
/// 1. Fetches RAG context from the backend /context endpoint
/// 2. Connects to Gemini Live API via WebSocket
/// 3. Streams microphone audio to Gemini
/// 4. Plays back Gemini's audio response
@riverpod
class TutorSessionController extends _$TutorSessionController {
  // SECURITY: API key must come from environment, never hardcoded in code.
  // For production: use --dart-define=GEMINI_API_KEY=... at build time.
  static const String _geminiApiKey = String.fromEnvironment(
    'GEMINI_API_KEY',
    defaultValue: '',
  );

  late GeminiLiveService _geminiService;
  final AudioRecorder _recorder = AudioRecorder();
  final AudioPlayer _player = AudioPlayer();

  StreamSubscription<SessionState>? _stateSub;
  StreamSubscription<Uint8List>? _audioSub;
  StreamSubscription<String>? _transcriptSub;
  StreamSubscription<List<int>>? _audioMicSub; // mic stream subscription

  @override
  TutorSessionState build() {
    _geminiService = GeminiLiveService(_geminiApiKey);
    ref.onDispose(() {
      _cleanup();
    });
    return const TutorSessionState();
  }

  /// Start a tutoring session for a given course topic.
  Future<void> startSession(String courseId, String topicId) async {
    // Guard: API key must be injected via --dart-define at build time
    if (_geminiApiKey.isEmpty) {
      state = state.copyWith(
        sessionState: SessionState.error,
        error:
            'Gemini API key not configured. Run with --dart-define=GEMINI_API_KEY=...',
      );
      return;
    }
    state = state.copyWith(sessionState: SessionState.connecting, error: null);

    try {
      // Step 1: Fetch RAG context from our Go backend
      final dio = ref.read(dioClientProvider);

      final readinessResponse = await dio.get(
        '/courses/$courseId/topics/$topicId/readiness',
      );
      final isReady = (readinessResponse.data['is_ready'] as bool?) ?? false;
      if (!isReady) {
        final message =
            (readinessResponse.data['message'] as String?) ??
            'Upload and validate at least one material before tutoring.';
        state = state.copyWith(
          sessionState: SessionState.error,
          error: message,
        );
        return;
      }

      final context = await _fetchContext(dio, courseId, topicId);

      // Step 2: Subscribe to the GeminiLiveService streams
      _stateSub = _geminiService.stateStream.listen((s) {
        state = state.copyWith(sessionState: s);
      });

      // Step 3: Play audio responses from Gemini
      _audioSub = _geminiService.audioOutputStream.listen((audioBytes) {
        // audioplayers can play raw PCM bytes
        _player.play(BytesSource(audioBytes));
      });

      // Step 4: Accumulate transcript text
      var fullTranscript = '';
      _transcriptSub = _geminiService.transcriptStream.listen((chunk) {
        fullTranscript += chunk;
        state = state.copyWith(transcript: fullTranscript);
      });

      // Step 5: Connect to Gemini Live with the RAG context
      await _geminiService.connect(context);

      // Step 6: Start microphone streaming
      await _startMicrophone();
    } catch (e) {
      debugPrint('[TutorSession] startSession error: $e');
      state = state.copyWith(
        sessionState: SessionState.error,
        error: 'Could not start session. Please try again.',
      );
    }
  }

  /// Stop the tutoring session and release all resources.
  Future<void> stopSession() async {
    await _stopMicrophone();
    await _geminiService.disconnect();
    state = state.copyWith(
      sessionState: SessionState.idle,
      isMicrophoneActive: false,
    );
  }

  Future<String> _fetchContext(Dio dio, String courseId, String topicId) async {
    try {
      final response = await dio.get(
        '/courses/$courseId/topics/$topicId/context',
      );
      return (response.data['context'] as String?) ?? '';
    } catch (e) {
      debugPrint(
        '[TutorSession] Could not fetch context: $e. Proceeding without RAG context.',
      );
      // Non-fatal: proceed with empty context — the Tutor still works without RAG
      return '';
    }
  }

  Future<void> _startMicrophone() async {
    final hasPermission = await _recorder.hasPermission();
    if (!hasPermission) {
      state = state.copyWith(
        error: 'Microphone permission denied. Please enable it in settings.',
        sessionState: SessionState.error,
      );
      return;
    }

    // PCM16 at 16kHz mono — the format Gemini Live expects
    const config = RecordConfig(
      encoder: AudioEncoder.pcm16bits,
      sampleRate: 16000,
      numChannels: 1,
    );

    final audioStream = await _recorder.startStream(config);
    state = state.copyWith(isMicrophoneActive: true);

    // Store subscription so it can be cancelled on stopSession()
    _audioMicSub = audioStream.listen((audioChunk) {
      _geminiService.sendAudioChunk(Uint8List.fromList(audioChunk));
    });
  }

  Future<void> _stopMicrophone() async {
    await _recorder.stop();
    state = state.copyWith(isMicrophoneActive: false);
  }

  Future<void> _cleanup() async {
    await _stateSub?.cancel();
    await _audioSub?.cancel();
    await _transcriptSub?.cancel();
    await _audioMicSub?.cancel(); // prevent mic stream leak
    await _recorder.dispose();
    await _player.dispose();
    await _geminiService.disconnect(); // await async disconnect
    _geminiService.dispose();
  }
}
