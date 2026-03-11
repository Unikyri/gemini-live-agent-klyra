import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';
import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

/// Represents the current state of the Gemini Live session.
enum SessionState { idle, connecting, active, speaking, error }

/// GeminiLiveService manages the real-time WebSocket connection to Gemini Live API.
///
/// Architecture:
/// - The mobile client connects directly to the Gemini Live API (BidiGenerateContent).
/// - Before connecting, it fetches the RAG context from our Go backend (/context endpoint).
/// - The context is injected into the system_instruction of the initial setup message.
///
/// This minimises latency — audio never passes through our backend.
class GeminiLiveService {
  static const _geminiWsUrl =
      'wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent';
  static const _model = 'models/gemini-2.0-flash-live-001';

  WebSocketChannel? _channel;
  StreamController<Uint8List>? _audioOutputController;
  final String _apiKey;

  /// Fires whenever the session state changes.
  final _stateController = StreamController<SessionState>.broadcast();
  Stream<SessionState> get stateStream => _stateController.stream;

  /// Emits audio PCM chunks received from Gemini (model's voice response).
  Stream<Uint8List> get audioOutputStream =>
      _audioOutputController?.stream ?? const Stream.empty();

  /// Fires whenever Gemini returns a text transcript chunk.
  final _transcriptController = StreamController<String>.broadcast();
  Stream<String> get transcriptStream => _transcriptController.stream;

  GeminiLiveService(this._apiKey);

  /// Connect to Gemini Live and send the initial setup (no RAG context — use sendContextUpdate to inject later).
  /// [topicTitles] is the list of topic titles for the course; used to build a short prompt asking the student to choose a topic.
  Future<void> connect({
    String? courseName,
    String? educationLevel,
    List<String> topicTitles = const [],
  }) async {
    _stateController.add(SessionState.connecting);
    _audioOutputController = StreamController<Uint8List>.broadcast();

    try {
      final uri = Uri.parse('$_geminiWsUrl?key=$_apiKey');
      _channel = WebSocketChannel.connect(uri);
      await _channel!.ready;

      final systemPrompt = _buildSystemPrompt(
        courseName: courseName,
        educationLevel: educationLevel,
        topicTitles: topicTitles,
      );

      // Send setup message with model and system instruction (no chunks — context loaded on demand)
      final setup = {
        'setup': {
          'model': _model,
          'system_instruction': {
            'parts': [
              {'text': systemPrompt}
            ]
          },
          'generation_config': {
            'response_modalities': ['AUDIO'],
            'speech_config': {
              'voice_config': {
                'prebuilt_voice_config': {'voice_name': 'Kore'}
              }
            }
          }
        }
      };
      _channel!.sink.add(jsonEncode(setup));

      // Listen for incoming messages
      _channel!.stream.listen(
        _handleMessage,
        onError: (error) {
          debugPrint('[GeminiLive] WebSocket error: $error');
          _stateController.add(SessionState.error);
        },
        onDone: () {
          debugPrint('[GeminiLive] WebSocket connection closed.');
          _stateController.add(SessionState.idle);
        },
      );

      _stateController.add(SessionState.active);
    } catch (e) {
      debugPrint('[GeminiLive] Connect failed: $e');
      _stateController.add(SessionState.error);
      rethrow;
    }
  }

  /// Sends a context update to the active session (e.g. after the user selects a topic).
  /// The text is sent as client_content so the model can use it as reference.
  void sendContextUpdate(String contextText) {
    if (_channel == null) return;
    final message = {
      'clientContent': {
        'parts': [
          {'text': '[CONTEXTO DEL TEMA SELECCIONADO]\n$contextText'}
        ],
        'turnComplete': true,
      }
    };
    _channel!.sink.add(jsonEncode(message));
  }

  /// Send a raw PCM audio chunk from the microphone to Gemini.
  void sendAudioChunk(Uint8List pcmBytes) {
    if (_channel == null) return;
    // Gemini Live expects base64-encoded PCM16 at 16000 Hz
    final b64 = base64Encode(pcmBytes);
    final message = {
      'realtimeInput': {
        'mediaChunks': [
          {
            'mimeType': 'audio/pcm;rate=16000',
            'data': b64,
          }
        ]
      }
    };
    _channel!.sink.add(jsonEncode(message));
  }

  /// Process a message received from Gemini Live.
  void _handleMessage(dynamic rawMessage) {
    try {
      final Map<String, dynamic> msg = jsonDecode(rawMessage as String);

      // Setup confirmation
      if (msg.containsKey('setupComplete')) {
        debugPrint('[GeminiLive] Setup confirmed by server.');
        return;
      }

      // Server content (audio or text)
      final serverContent = msg['serverContent'] as Map<String, dynamic>?;
      if (serverContent == null) return;

      final modelTurn = serverContent['modelTurn'] as Map<String, dynamic>?;
      if (modelTurn == null) return;

      final parts = modelTurn['parts'] as List<dynamic>? ?? [];
      for (final part in parts) {
        final partMap = part as Map<String, dynamic>;

        // Audio response from model
        final inlineData = partMap['inlineData'] as Map<String, dynamic>?;
        if (inlineData != null) {
          final audioData = base64Decode(inlineData['data'] as String);
          _audioOutputController?.add(Uint8List.fromList(audioData));
          _stateController.add(SessionState.speaking);
        }

        // Text transcript from model
        final text = partMap['text'] as String?;
        if (text != null && text.isNotEmpty) {
          _transcriptController.add(text);
        }
      }

      // Turn completion signal
      if (serverContent['turnComplete'] == true) {
        _stateController.add(SessionState.active);
      }
    } catch (e) {
      debugPrint('[GeminiLive] Failed to parse server message: $e');
    }
  }

  String _buildSystemPrompt({
    String? courseName,
    String? educationLevel,
    List<String> topicTitles = const [],
  }) {
    final courseLine = (courseName != null && courseName.isNotEmpty)
        ? 'This course is "$courseName"'
        : 'This is a course';
    final levelLine = (educationLevel != null && educationLevel.isNotEmpty)
        ? ' (level: $educationLevel).'
        : '.';
    final topicList = topicTitles.isNotEmpty
        ? topicTitles.asMap().entries.map((e) => '${e.key + 1}. ${e.value}').join('\n')
        : '(none)';
    return '''You are Klyra, an enthusiastic, patient, and encouraging AI tutor.
Your goal is to help the student understand their course material through natural conversation.
Speak clearly and at an appropriate pace.
Ask questions to check understanding.
Celebrate correct answers and gently correct mistakes.

$courseLine$levelLine
Available topics in this course:
$topicList

Ask the student which topic they want to discuss. When they choose one, you will receive the relevant context.
Until then, you can discuss the course structure and help them choose a topic.''';
  }

  /// Disconnect from Gemini Live and clean up resources.
  Future<void> disconnect() async {
    await _channel?.sink.close();
    _channel = null;
    await _audioOutputController?.close();
    _audioOutputController = null;
    _stateController.add(SessionState.idle);
  }

  void dispose() {
    disconnect();
    _stateController.close();
    _transcriptController.close();
  }
}
