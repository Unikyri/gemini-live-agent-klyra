import 'dart:async';
import 'dart:math' as math;
import 'dart:typed_data';

/// Tracks normalized RMS amplitude (0.0–1.0) from PCM16 LE audio.
class AudioAmplitudeTracker {
  final int sampleRate;
  final Duration window;

  final _controller = StreamController<double>.broadcast();
  Stream<double> get amplitudeStream => _controller.stream;

  Uint8List _buffer = Uint8List(0);

  AudioAmplitudeTracker({
    this.sampleRate = 24000,
    this.window = const Duration(milliseconds: 50),
  });

  void processAudioChunk(Uint8List pcm16Data) {
    if (pcm16Data.isEmpty) return;
    _buffer = Uint8List.fromList(<int>[..._buffer, ...pcm16Data]);

    final windowSamples = (sampleRate * window.inMilliseconds / 1000).ceil();
    final windowBytes = windowSamples * 2;
    while (_buffer.lengthInBytes >= windowBytes) {
      final frame = _buffer.sublist(0, windowBytes);
      _buffer = _buffer.sublist(windowBytes);
      _controller.add(_computeRmsNormalized(frame));
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

  Future<void> dispose() async {
    await _controller.close();
  }
}

