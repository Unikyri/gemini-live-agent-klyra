// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'tutor_session_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// TutorSessionController manages the full lifecycle of a Klyra tutoring session:
/// 1. Fetches RAG context from the backend /context endpoint
/// 2. Connects to Gemini Live API via WebSocket
/// 3. Streams microphone audio to Gemini
/// 4. Plays back Gemini's audio response

@ProviderFor(TutorSessionController)
final tutorSessionControllerProvider = TutorSessionControllerProvider._();

/// TutorSessionController manages the full lifecycle of a Klyra tutoring session:
/// 1. Fetches RAG context from the backend /context endpoint
/// 2. Connects to Gemini Live API via WebSocket
/// 3. Streams microphone audio to Gemini
/// 4. Plays back Gemini's audio response
final class TutorSessionControllerProvider
    extends $NotifierProvider<TutorSessionController, TutorSessionState> {
  /// TutorSessionController manages the full lifecycle of a Klyra tutoring session:
  /// 1. Fetches RAG context from the backend /context endpoint
  /// 2. Connects to Gemini Live API via WebSocket
  /// 3. Streams microphone audio to Gemini
  /// 4. Plays back Gemini's audio response
  TutorSessionControllerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'tutorSessionControllerProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$tutorSessionControllerHash();

  @$internal
  @override
  TutorSessionController create() => TutorSessionController();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(TutorSessionState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<TutorSessionState>(value),
    );
  }
}

String _$tutorSessionControllerHash() =>
    r'73a03d2a10139a7c9d76069484e6f8782d4cf032';

/// TutorSessionController manages the full lifecycle of a Klyra tutoring session:
/// 1. Fetches RAG context from the backend /context endpoint
/// 2. Connects to Gemini Live API via WebSocket
/// 3. Streams microphone audio to Gemini
/// 4. Plays back Gemini's audio response

abstract class _$TutorSessionController extends $Notifier<TutorSessionState> {
  TutorSessionState build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<TutorSessionState, TutorSessionState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<TutorSessionState, TutorSessionState>,
              TutorSessionState,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
