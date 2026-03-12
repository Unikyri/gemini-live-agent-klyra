import 'dart:async';
import 'dart:math' as math;
import 'dart:typed_data';

/// Voice Activity Detection (VAD) interface.
abstract class VadDetector {
  Stream<bool> get isSpeakingStream;
  void processAudioChunk(Uint8List pcm16Data);
  void dispose();
}

/// Simple RMS-based VAD over PCM16 little-endian mono audio.
///
/// Emits `true` after RMS stays above [threshold] for at least [minDuration].
/// Emits `false` when RMS drops below [threshold].
class RmsVadDetector implements VadDetector {
  final double threshold;
  final Duration minDuration;
  final int sampleRate;

  final _controller = StreamController<bool>.broadcast();
  @override
  Stream<bool> get isSpeakingStream => _controller.stream;

  bool _isSpeaking = false;
  int _aboveThresholdSamples = 0;

  RmsVadDetector({
    this.threshold = 0.03,
    this.minDuration = const Duration(milliseconds: 250),
    this.sampleRate = 16000,
  });

  @override
  void processAudioChunk(Uint8List pcm16Data) {
    if (pcm16Data.isEmpty) return;
    final rms = _computeRmsNormalized(pcm16Data);

    final samplesInChunk = pcm16Data.lengthInBytes ~/ 2;
    final minSamples = (sampleRate * minDuration.inMilliseconds / 1000).ceil();

    if (rms >= threshold) {
      _aboveThresholdSamples += samplesInChunk;
      if (!_isSpeaking && _aboveThresholdSamples >= minSamples) {
        _isSpeaking = true;
        _controller.add(true);
      }
    } else {
      _aboveThresholdSamples = 0;
      if (_isSpeaking) {
        _isSpeaking = false;
        _controller.add(false);
      }
    }
  }

  double _computeRmsNormalized(Uint8List pcm16Le) {
    final byteData = ByteData.sublistView(pcm16Le);
    final n = pcm16Le.lengthInBytes ~/ 2;
    if (n <= 0) return 0;

    double sumSquares = 0;
    for (var i = 0; i < n; i++) {
      final s = byteData.getInt16(i * 2, Endian.little);
      final x = s / 32768.0;
      sumSquares += x * x;
    }
    final mean = sumSquares / n;
    return math.sqrt(mean).clamp(0.0, 1.0);
  }

  @override
  void dispose() {
    _controller.close();
  }
}

