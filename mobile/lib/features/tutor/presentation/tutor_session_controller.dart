import 'dart:async';
import 'dart:typed_data';
import 'package:audioplayers/audioplayers.dart';
import 'package:flutter/foundation.dart';
import 'package:record/record.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:riverpod/riverpod.dart';
import 'package:klyra/core/network/dio_client.dart';
import 'package:klyra/features/course/data/course_repository.dart';
import 'package:klyra/features/tutor/data/gemini_live_service.dart';

part 'tutor_session_controller.g.dart';

typedef GeminiLiveServiceFactory = GeminiLiveService Function(String apiKey);

final geminiLiveServiceFactoryProvider = Provider<GeminiLiveServiceFactory>(
  (ref) => (apiKey) => GeminiLiveService(apiKey),
);

/// Holds the complete state of the tutor session UI.
class TutorSessionState {
  final SessionState sessionState;
  final String transcript;
  final String? error;
  final bool isMicrophoneActive;
  final String? currentTopicId;
  final Set<String> loadedTopicIds;
  final bool isLoadingContext;
  final bool hasCurrentTopicMaterials;

  const TutorSessionState({
    this.sessionState = SessionState.idle,
    this.transcript = '',
    this.error,
    this.isMicrophoneActive = false,
    this.currentTopicId,
    Set<String>? loadedTopicIds,
    this.isLoadingContext = false,
    this.hasCurrentTopicMaterials = true,
  }) : loadedTopicIds = loadedTopicIds ?? const {};

  TutorSessionState copyWith({
    SessionState? sessionState,
    String? transcript,
    String? error,
    bool? isMicrophoneActive,
    String? currentTopicId,
    Set<String>? loadedTopicIds,
    bool? isLoadingContext,
    bool? hasCurrentTopicMaterials,
  }) {
    return TutorSessionState(
      sessionState: sessionState ?? this.sessionState,
      transcript: transcript ?? this.transcript,
      error: error,
      isMicrophoneActive: isMicrophoneActive ?? this.isMicrophoneActive,
      currentTopicId: currentTopicId ?? this.currentTopicId,
      loadedTopicIds: loadedTopicIds ?? this.loadedTopicIds,
      isLoadingContext: isLoadingContext ?? this.isLoadingContext,
      hasCurrentTopicMaterials:
          hasCurrentTopicMaterials ?? this.hasCurrentTopicMaterials,
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
  AudioRecorder? _recorder;
  AudioPlayer? _player;

  StreamSubscription<SessionState>? _stateSub;
  StreamSubscription<Uint8List>? _audioSub;
  StreamSubscription<String>? _transcriptSub;
  StreamSubscription<List<int>>? _audioMicSub; // mic stream subscription

  @override
  TutorSessionState build() {
    final serviceFactory = ref.read(geminiLiveServiceFactoryProvider);
    _geminiService = serviceFactory(_geminiApiKey);
    ref.onDispose(() {
      _cleanup();
    });
    return const TutorSessionState();
  }

  /// Start a tutoring session at course level. [topicId] is optional; if provided, that topic's context is loaded after connecting.
  Future<void> startSession(String courseId, {String? topicId}) async {
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
      final repo = ref.read(courseRepositoryProvider);
      final course = await repo.getCourse(courseId);
      final topicTitles = course.topics.map((t) => t.title).toList();

      _stateSub = _geminiService.stateStream.listen((s) {
        state = state.copyWith(sessionState: s);
      });
      _audioSub = _geminiService.audioOutputStream.listen((audioBytes) {
        _player?.play(BytesSource(audioBytes));
      });
      var fullTranscript = '';
      _transcriptSub = _geminiService.transcriptStream.listen((chunk) {
        fullTranscript += chunk;
        state = state.copyWith(transcript: fullTranscript);
      });

      await _geminiService.connect(
        courseName: course.name,
        educationLevel: course.educationLevel,
        topicTitles: topicTitles,
      );

      if (topicId != null && topicId.isNotEmpty) {
        await loadTopicContext(courseId, topicId);
      }

      _recorder ??= AudioRecorder();
      _player ??= AudioPlayer();
      await _startMicrophone();
    } catch (e) {
      debugPrint('[TutorSession] startSession error: $e');
      state = state.copyWith(
        sessionState: SessionState.error,
        error: 'Could not start session. Please try again.',
      );
    }
  }

  /// Load context for a specific topic and send it to the active Gemini session.
  Future<void> loadTopicContext(String courseId, String topicId) async {
    if (state.loadedTopicIds.contains(topicId)) {
      // Topic context already loaded previously; just switch selection.
      state = state.copyWith(currentTopicId: topicId);
      return;
    }
    state = state.copyWith(isLoadingContext: true);
    try {
      final dio = ref.read(dioClientProvider);
      final response = await dio.get(
        '/courses/$courseId/topics/$topicId/context',
      );
      final contextText = (response.data['context'] as String?) ?? '';
      final hasMaterials =
          (response.data['has_materials'] as bool?) ?? contextText.isNotEmpty;
      final message = (response.data['message'] as String?) ?? '';

      if (contextText.isNotEmpty) {
        _geminiService.sendContextUpdate(contextText);
      } else {
        // Zero-material: build minimal context from topic title so tutor knows the intent.
        final repo = ref.read(courseRepositoryProvider);
        final course = await repo.getCourse(courseId);
        final topicTitle = course.topics
                .where((t) => t.id == topicId)
                .firstOrNull
                ?.title ??
            topicId;
        final minimalContext = StringBuffer()
          ..writeln('[Contexto actualizado — Tema: $topicTitle]')
          ..writeln(
              'El estudiante quiere hablar del tema: "$topicTitle". No hay material de referencia para este tema.')
          ..writeln(
              'Usa tu conocimiento para guiar la conversación de forma útil.')
          ..write(
              'Si el estudiante quiere respuestas más precisas, puede subir material de estudio.');
        if (message.isNotEmpty) {
          minimalContext
              .writeln('\n\nNota del sistema: $message');
        }
        _geminiService.sendContextUpdate(minimalContext.toString());
      }

      state = state.copyWith(
        currentTopicId: topicId,
        loadedTopicIds: {...state.loadedTopicIds, topicId},
        isLoadingContext: false,
        hasCurrentTopicMaterials: hasMaterials,
      );
    } catch (e) {
      debugPrint('[TutorSession] loadTopicContext error: $e');
      state = state.copyWith(isLoadingContext: false);
    }
  }

  /// Load course-level (truncated) context and send it to the active Gemini session.
  Future<void> loadCourseContext(String courseId) async {
    state = state.copyWith(isLoadingContext: true);
    try {
      final repo = ref.read(courseRepositoryProvider);
      final data = await repo.fetchCourseContext(courseId);
      final contextText = (data['context'] as String?) ?? '';
      _geminiService.sendContextUpdate(contextText);
      state = state.copyWith(
        currentTopicId: null,
        isLoadingContext: false,
      );
    } catch (e) {
      debugPrint('[TutorSession] loadCourseContext error: $e');
      state = state.copyWith(isLoadingContext: false);
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

  Future<void> _startMicrophone() async {
    final recorder = _recorder;
    if (recorder == null) return;

    final hasPermission = await recorder.hasPermission();
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

    final audioStream = await recorder.startStream(config);
    state = state.copyWith(isMicrophoneActive: true);

    // Store subscription so it can be cancelled on stopSession()
    _audioMicSub = audioStream.listen((audioChunk) {
      _geminiService.sendAudioChunk(Uint8List.fromList(audioChunk));
    });
  }

  Future<void> _stopMicrophone() async {
    await _recorder?.stop();
    state = state.copyWith(isMicrophoneActive: false);
  }

  Future<void> _cleanup() async {
    await _stateSub?.cancel();
    await _audioSub?.cancel();
    await _transcriptSub?.cancel();
    await _audioMicSub?.cancel(); // prevent mic stream leak
    await _recorder?.dispose();
    await _player?.dispose();
    await _geminiService.disconnect(); // await async disconnect
    _geminiService.dispose();
  }
}
