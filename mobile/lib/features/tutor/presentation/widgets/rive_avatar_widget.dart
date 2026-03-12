import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:rive/rive.dart';

enum TutorAvatarState { idle, speaking, listening, thinking, reconnecting }

class RiveAvatarWidget extends StatefulWidget {
  final Stream<double> amplitudeStream;
  final TutorAvatarState avatarState;
  final String assetPath;

  const RiveAvatarWidget({
    super.key,
    required this.amplitudeStream,
    required this.avatarState,
    this.assetPath = 'assets/rive/tutor_avatar.riv',
  });

  @override
  State<RiveAvatarWidget> createState() => _RiveAvatarWidgetState();
}

class _RiveAvatarWidgetState extends State<RiveAvatarWidget> {
  StreamSubscription<double>? _ampSub;

  Artboard? _artboard;
  SMINumber? _mouthOpen;
  StateMachineController? _smController;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
    _ampSub = widget.amplitudeStream.listen((amp) {
      _mouthOpen?.value = amp.clamp(0.0, 1.0);
    });
  }

  @override
  void didUpdateWidget(covariant RiveAvatarWidget oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.avatarState != widget.avatarState) {
      _applyAvatarState(widget.avatarState);
    }
  }

  Future<void> _load() async {
    try {
      final ByteData data = await rootBundle.load(widget.assetPath);
      final file = RiveFile.import(data);
      final artboard = file.mainArtboard;

      final sm = StateMachineController.fromArtboard(
        artboard,
        'MainStateMachine',
      );
      if (sm != null) {
        artboard.addController(sm);
        _smController = sm;
      }

      final mouth = sm?.findInput<double>('mouthOpen');
      if (mouth is SMINumber) _mouthOpen = mouth;

      if (!mounted) return;
      setState(() => _artboard = artboard);
      _applyAvatarState(widget.avatarState);
    } catch (_) {
      if (!mounted) return;
      setState(() => _artboard = null);
    }
  }

  void _applyAvatarState(TutorAvatarState state) {
    final ctrl = _smController;
    if (ctrl == null) return;

    // Best-effort wiring: if the .riv doesn't provide inputs, this is a no-op.
    final stateName = switch (state) {
      TutorAvatarState.idle => 'idle',
      TutorAvatarState.speaking => 'speaking',
      TutorAvatarState.listening => 'listening',
      TutorAvatarState.thinking => 'thinking',
      TutorAvatarState.reconnecting => 'reconnecting',
    };

    final boolInput = ctrl.findInput<bool>(stateName);
    if (boolInput is SMIBool) {
      boolInput.value = true;
    }
  }

  @override
  void dispose() {
    _ampSub?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_artboard == null) {
      return const Center(
        child: Icon(Icons.smart_toy_rounded, size: 80, color: Color(0xFF6C63FF)),
      );
    }
    return Rive(artboard: _artboard!);
  }
}

