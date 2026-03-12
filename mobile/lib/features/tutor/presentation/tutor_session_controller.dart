import 'dart:async';
import 'package:flutter/services.dart';
import 'dart:typed_data';
import 'package:audioplayers/audioplayers.dart';
import 'package:flutter/foundation.dart';
import 'package:record/record.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:riverpod/riverpod.dart';
import 'package:klyra/core/network/dio_client.dart';
import 'package:klyra/features/course/data/course_repository.dart';
import 'package:klyra/features/tutor/data/audio_amplitude_tracker.dart';
import 'package:klyra/features/tutor/data/gemini_live_service.dart';
import 'package:klyra/features/tutor/data/vad_detector.dart';
import 'package:klyra/features/tutor/presentation/widgets/rive_avatar_widget.dart';

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
  final bool isBargeInActive;
  final String backgroundContextType;
  final TutorAvatarState avatarState;

  const TutorSessionState({
    this.sessionState = SessionState.idle,
    this.transcript = '',
    this.error,
    this.isMicrophoneActive = false,
    this.currentTopicId,
    Set<String>? loadedTopicIds,
    this.isLoadingContext = false,
    this.hasCurrentTopicMaterials = true,
    this.isBargeInActive = false,
    this.backgroundContextType = 'default',
    this.avatarState = TutorAvatarState.idle,
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
    bool? isBargeInActive,
    String? backgroundContextType,
    TutorAvatarState? avatarState,
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
      isBargeInActive: isBargeInActive ?? this.isBargeInActive,
      backgroundContextType: backgroundContextType ?? this.backgroundContextType,
      avatarState: avatarState ?? this.avatarState,
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

  static const bool _ffLearningProfile = bool.fromEnvironment(
    'FF_LEARNING_PROFILE',
    defaultValue: false,
  );
  static const bool _ffBargeIn = bool.fromEnvironment(
    'FF_BARGE_IN',
    defaultValue: false,
  );
  static const bool _ffAvatarRive = bool.fromEnvironment(
    'FF_AVATAR_RIVE',
    defaultValue: false,
  );
  static const bool _ffDynamicBackgrounds = bool.fromEnvironment(
    'FF_DYNAMIC_BACKGROUNDS',
    defaultValue: false,
  );
  static const bool _ffCameraSnapshot = bool.fromEnvironment(
    'FF_CAMERA_SNAPSHOT',
    defaultValue: false,
  );

  static const int _learningProfileUpdateInterval = int.fromEnvironment(
    'LEARNING_PROFILE_UPDATE_INTERVAL',
    defaultValue: 10,
  );

  late GeminiLiveService _geminiService;
  AudioRecorder? _recorder;
  AudioPlayer? _player;
  VadDetector? _vad;
  StreamSubscription<bool>? _vadSub;
  final _amplitudeTracker = AudioAmplitudeTracker();
  final _amplitudeController = StreamController<double>.broadcast();
  Stream<double> get amplitudeStream => _amplitudeController.stream;

  StreamSubscription<String>? _backgroundSub;
  StreamSubscription<double>? _amplitudeSub;

  StreamSubscription<SessionState>? _stateSub;
  StreamSubscription<Uint8List>? _audioSub;
  StreamSubscription<String>? _transcriptSub;
  StreamSubscription<List<int>>? _audioMicSub; // mic stream subscription

  final List<String> _recentTranscriptChunks = [];
  int _profileChunkCounter = 0;
  final _audioSessionChannel = const MethodChannel('klyra/audio_session');
  bool _micDesired = false;

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
    _micDesired = true;

    try {
      final repo = ref.read(courseRepositoryProvider);
      final course = await repo.getCourse(courseId);
      final topicTitles = course.topics.map((t) => t.title).toList();

      _stateSub = _geminiService.stateStream.listen((s) {
        state = state.copyWith(sessionState: s);
        state = state.copyWith(
          avatarState: switch (s) {
            SessionState.speaking => TutorAvatarState.speaking,
            SessionState.reconnecting => TutorAvatarState.reconnecting,
            SessionState.connecting => TutorAvatarState.thinking,
            SessionState.error => TutorAvatarState.thinking,
            _ => TutorAvatarState.idle,
          },
        );
        if (s == SessionState.reconnecting) {
          unawaited(_stopMicrophone());
        }
        if (s == SessionState.active && _micDesired && !state.isMicrophoneActive) {
          unawaited(_startMicrophone());
        }
      });
      _audioSub = _geminiService.audioOutputStream.listen((audioBytes) {
        _player?.play(BytesSource(audioBytes));
        _amplitudeTracker.processAudioChunk(audioBytes);
      });
      _backgroundSub = _geminiService.backgroundContextStream.listen((ctx) {
        if (_ffDynamicBackgrounds) {
          state = state.copyWith(backgroundContextType: ctx);
        }
      });
      _amplitudeSub ??=
          _amplitudeTracker.amplitudeStream.listen(_amplitudeController.add);
      var fullTranscript = '';
      _transcriptSub = _geminiService.transcriptStream.listen((chunk) {
        fullTranscript += chunk;
        state = state.copyWith(transcript: fullTranscript);
        _trackAndMaybeUpdateLearningProfile(chunk);
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
      await _configureAec();
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
    await _flushLearningProfileUpdate();
    _micDesired = false;
    await _stopMicrophone();
    await _resetAec();
    await _geminiService.disconnect();
    state = state.copyWith(
      sessionState: SessionState.idle,
      isMicrophoneActive: false,
    );
  }

  void _trackAndMaybeUpdateLearningProfile(String chunk) {
    if (!_ffLearningProfile) return;
    final cleaned = chunk.trim();
    if (cleaned.isEmpty) return;
    _recentTranscriptChunks.add(cleaned);
    if (_recentTranscriptChunks.length > 50) {
      _recentTranscriptChunks.removeRange(
        0,
        _recentTranscriptChunks.length - 50,
      );
    }
    _profileChunkCounter++;
    if (_profileChunkCounter >= _learningProfileUpdateInterval) {
      _profileChunkCounter = 0;
      unawaited(_sendLearningProfileUpdate(_recentTranscriptChunks.takeLast(20)));
    }
  }

  Future<void> _flushLearningProfileUpdate() async {
    if (!_ffLearningProfile) return;
    final batch = _recentTranscriptChunks.takeLast(20);
    if (batch.isEmpty) return;
    await _sendLearningProfileUpdate(batch);
    _recentTranscriptChunks.clear();
    _profileChunkCounter = 0;
  }

  Future<void> _sendLearningProfileUpdate(List<String> recent) async {
    try {
      final dio = ref.read(dioClientProvider);
      await dio.post(
        '/users/me/learning-profile/update',
        data: {'recent_messages': recent},
      );
    } catch (e) {
      debugPrint('[LearningProfile] update failed: $e');
    }
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

    _vad ??= RmsVadDetector();
    _vadSub ??= _vad!.isSpeakingStream.listen((isSpeaking) {
      if (!_ffBargeIn) return;
      if (isSpeaking && state.sessionState == SessionState.speaking) {
        unawaited(_triggerBargeIn());
      }
    });

    // Store subscription so it can be cancelled on stopSession()
    _audioMicSub = audioStream.listen((audioChunk) {
      _geminiService.sendAudioChunk(Uint8List.fromList(audioChunk));
      _vad?.processAudioChunk(Uint8List.fromList(audioChunk));
    });
  }

  Future<void> _stopMicrophone() async {
    await _audioMicSub?.cancel();
    _audioMicSub = null;
    await _recorder?.stop();
    state = state.copyWith(isMicrophoneActive: false);
  }

  Future<void> sendSnapshotToTutor(String base64Jpeg) async {
    if (!_ffCameraSnapshot) return;
    _geminiService.sendImageData(
      base64Jpeg: base64Jpeg,
      promptText: 'Mira mis apuntes y explícame lo que ves',
    );
  }

  Future<void> _triggerBargeIn() async {
    state = state.copyWith(
      isBargeInActive: true,
      avatarState: TutorAvatarState.listening,
    );

    try {
      await _player?.stop();
      await _player?.release();
      _player = AudioPlayer();
    } catch (e) {
      debugPrint('[BargeIn] stop/release failed: $e');
    } finally {
      Future<void>.delayed(const Duration(milliseconds: 600), () {
        if (!ref.mounted) return;
        state = state.copyWith(isBargeInActive: false);
      });
    }
  }

  Future<void> _configureAec() async {
    try {
      await _audioSessionChannel.invokeMethod('setVoiceChatMode');
    } catch (e) {
      debugPrint('[AEC] configure failed: $e');
    }
  }

  Future<void> _resetAec() async {
    try {
      await _audioSessionChannel.invokeMethod('resetAudioMode');
    } catch (e) {
      debugPrint('[AEC] reset failed: $e');
    }
  }

  Future<void> _cleanup() async {
    await _stateSub?.cancel();
    await _audioSub?.cancel();
    await _transcriptSub?.cancel();
    await _audioMicSub?.cancel(); // prevent mic stream leak
    await _vadSub?.cancel();
    _vad?.dispose();
    await _backgroundSub?.cancel();
    await _amplitudeTracker.dispose();
    await _amplitudeSub?.cancel();
    await _amplitudeController.close();
    await _recorder?.dispose();
    await _player?.dispose();
    await _geminiService.disconnect(); // await async disconnect
    _geminiService.dispose();
  }
}

extension<T> on List<T> {
  List<T> takeLast(int n) {
    if (n <= 0) return const [];
    if (length <= n) return List<T>.from(this);
    return sublist(length - n);
  }
}
