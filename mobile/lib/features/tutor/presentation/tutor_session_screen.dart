import 'dart:math' as math;
import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:klyra/features/course/domain/course_models.dart';
import 'package:klyra/features/course/presentation/course_controller.dart';
import 'package:klyra/features/tutor/data/gemini_live_service.dart';
import 'package:klyra/features/tutor/presentation/tutor_session_controller.dart';

class TutorSessionScreen extends ConsumerStatefulWidget {
  final String courseId;
  final String topicId;

  const TutorSessionScreen({
    super.key,
    required this.courseId,
    required this.topicId,
  });

  @override
  ConsumerState<TutorSessionScreen> createState() => _TutorSessionScreenState();
}

class _TutorSessionScreenState extends ConsumerState<TutorSessionScreen>
    with TickerProviderStateMixin {
  late AnimationController _pulseController;
  late AnimationController _waveController;
  late Animation<double> _pulseAnim;

  @override
  void initState() {
    super.initState();
    _pulseController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1200),
    )..repeat(reverse: true);
    _waveController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 2000),
    )..repeat();
    _pulseAnim = Tween<double>(begin: 0.95, end: 1.05).animate(
      CurvedAnimation(parent: _pulseController, curve: Curves.easeInOut),
    );
  }

  @override
  void dispose() {
    _pulseController.dispose();
    _waveController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final sessionState = ref.watch(tutorSessionControllerProvider);
    final coursesAsync = ref.watch(courseControllerProvider);
    final theme = Theme.of(context);

    final Course? course = coursesAsync.valueOrNull
        ?.where((c) => c.id == widget.courseId)
        .firstOrNull;
    final Topic? topic = course?.topics
        .where((t) => t.id == widget.topicId)
        .firstOrNull;

    final bool isActive = sessionState.sessionState == SessionState.active ||
        sessionState.sessionState == SessionState.speaking;
    final bool isConnecting =
        sessionState.sessionState == SessionState.connecting;
    final bool isSpeaking =
        sessionState.sessionState == SessionState.speaking;

    return Scaffold(
      backgroundColor: const Color(0xFF0A0A1A),
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        elevation: 0,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(course?.name ?? 'Tutor Session',
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.bold)),
            if (topic != null)
              Text(topic.title,
                  style: theme.textTheme.labelSmall
                      ?.copyWith(color: Colors.white38)),
          ],
        ),
        actions: [
          if (isActive)
            TextButton.icon(
              onPressed: () =>
                  ref.read(tutorSessionControllerProvider.notifier).stopSession(),
              icon: const Icon(Icons.stop_circle_outlined, color: Colors.redAccent),
              label: const Text('End', style: TextStyle(color: Colors.redAccent)),
            ),
        ],
      ),
      body: SafeArea(
        child: Column(
          children: [
            // --- Avatar Display Area ---
            Expanded(
              flex: 6,
              child: _AvatarDisplay(
                avatarUrl: course?.avatarModelUrl,
                isSpeaking: isSpeaking,
                pulseAnim: _pulseAnim,
                waveController: _waveController,
                sessionState: sessionState.sessionState,
              ),
            ),

            // --- Transcript Area ---
            if (sessionState.transcript.isNotEmpty)
              Expanded(
                flex: 2,
                child: _TranscriptPanel(text: sessionState.transcript),
              ),

            // --- Error Message ---
            if (sessionState.error != null)
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 8),
                child: Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.redAccent.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: Colors.redAccent.withOpacity(0.4)),
                  ),
                  child: Text(
                    sessionState.error!,
                    style: theme.textTheme.bodySmall
                        ?.copyWith(color: Colors.redAccent),
                    textAlign: TextAlign.center,
                  ),
                ),
              ),

            // --- Mic Button ---
            Padding(
              padding: const EdgeInsets.only(bottom: 40, top: 16),
              child: _MicButton(
                isActive: isActive,
                isConnecting: isConnecting,
                isMicOn: sessionState.isMicrophoneActive,
                onStart: () => ref
                    .read(tutorSessionControllerProvider.notifier)
                    .startSession(widget.courseId, widget.topicId),
                onStop: () => ref
                    .read(tutorSessionControllerProvider.notifier)
                    .stopSession(),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// The animated avatar display area.
class _AvatarDisplay extends StatelessWidget {
  final String? avatarUrl;
  final bool isSpeaking;
  final Animation<double> pulseAnim;
  final AnimationController waveController;
  final SessionState sessionState;

  const _AvatarDisplay({
    required this.avatarUrl,
    required this.isSpeaking,
    required this.pulseAnim,
    required this.waveController,
    required this.sessionState,
  });

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Stack(
        alignment: Alignment.center,
        children: [
          // Animated glow rings when speaking
          if (isSpeaking)
            AnimatedBuilder(
              animation: waveController,
              builder: (_, __) {
                return CustomPaint(
                  painter: _WaveRingPainter(waveController.value),
                  size: const Size(320, 320),
                );
              },
            ),

          // Avatar image with pulse animation
          AnimatedBuilder(
            animation: pulseAnim,
            builder: (_, child) => Transform.scale(
              scale: isSpeaking ? pulseAnim.value : 1.0,
              child: child,
            ),
            child: Container(
              width: 220,
              height: 220,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                gradient: RadialGradient(
                  colors: [
                    const Color(0xFF6C63FF).withOpacity(0.3),
                    const Color(0xFF0A0A1A),
                  ],
                ),
                border: Border.all(
                  color: const Color(0xFF6C63FF).withOpacity(isSpeaking ? 0.8 : 0.3),
                  width: 2,
                ),
                boxShadow: isSpeaking
                    ? [
                        BoxShadow(
                          color: const Color(0xFF6C63FF).withOpacity(0.4),
                          blurRadius: 40,
                          spreadRadius: 10,
                        )
                      ]
                    : [],
              ),
              clipBehavior: Clip.antiAlias,
              child: avatarUrl != null
                  ? CachedNetworkImage(
                      imageUrl: avatarUrl!,
                      fit: BoxFit.contain,
                      errorWidget: (_, __, ___) =>
                          _DefaultAvatarIcon(sessionState: sessionState),
                    )
                  : _DefaultAvatarIcon(sessionState: sessionState),
            ),
          ),

          // Status label
          Positioned(
            bottom: 0,
            child: _StatusBadge(sessionState: sessionState),
          ),
        ],
      ),
    );
  }
}

class _DefaultAvatarIcon extends StatelessWidget {
  final SessionState sessionState;
  const _DefaultAvatarIcon({required this.sessionState});

  @override
  Widget build(BuildContext context) {
    return Container(
      color: const Color(0xFF1A1A2E),
      child: Center(
        child: Icon(
          Icons.smart_toy_rounded,
          size: 80,
          color: const Color(0xFF6C63FF).withOpacity(
            sessionState == SessionState.idle ? 0.4 : 0.9,
          ),
        ),
      ),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  final SessionState sessionState;
  const _StatusBadge({required this.sessionState});

  @override
  Widget build(BuildContext context) {
    final (label, color) = switch (sessionState) {
      SessionState.idle => ('Tap to start', Colors.white38),
      SessionState.connecting => ('Connecting...', Colors.orangeAccent),
      SessionState.active => ('Listening...', Colors.greenAccent),
      SessionState.speaking => ('Klyra is speaking', const Color(0xFF6C63FF)),
      SessionState.error => ('Error', Colors.redAccent),
    };

    return Container(
      margin: const EdgeInsets.only(top: 12),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      decoration: BoxDecoration(
        color: color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: color.withOpacity(0.4)),
      ),
      child: Text(label,
          style: TextStyle(color: color, fontSize: 12, fontWeight: FontWeight.w600)),
    );
  }
}

class _TranscriptPanel extends StatelessWidget {
  final String text;
  const _TranscriptPanel({required this.text});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.05),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: Colors.white.withOpacity(0.08)),
      ),
      child: SingleChildScrollView(
        reverse: true,
        child: Text(
          text,
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: Colors.white70,
                height: 1.6,
              ),
        ),
      ),
    );
  }
}

class _MicButton extends StatelessWidget {
  final bool isActive;
  final bool isConnecting;
  final bool isMicOn;
  final VoidCallback onStart;
  final VoidCallback onStop;

  const _MicButton({
    required this.isActive,
    required this.isConnecting,
    required this.isMicOn,
    required this.onStart,
    required this.onStop,
  });

  @override
  Widget build(BuildContext context) {
    if (isConnecting) {
      return const SizedBox(
        width: 80,
        height: 80,
        child: CircularProgressIndicator(color: Color(0xFF6C63FF), strokeWidth: 3),
      );
    }

    return GestureDetector(
      onTap: isActive ? onStop : onStart,
      child: Container(
        width: 80,
        height: 80,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: isActive
                ? [const Color(0xFFFF4081), const Color(0xFFFF6E40)]
                : [const Color(0xFF6C63FF), const Color(0xFF3D5AFE)],
          ),
          boxShadow: [
            BoxShadow(
              color: (isActive ? Colors.redAccent : const Color(0xFF6C63FF))
                  .withOpacity(0.5),
              blurRadius: 24,
              spreadRadius: 4,
            )
          ],
        ),
        child: Icon(
          isActive
              ? (isMicOn ? Icons.mic : Icons.mic_off)
              : Icons.mic_none_rounded,
          size: 36,
          color: Colors.white,
        ),
      ),
    );
  }
}

/// Custom painter that draws animated expanding wave rings.
class _WaveRingPainter extends CustomPainter {
  final double progress;
  _WaveRingPainter(this.progress);

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.5;

    for (int i = 0; i < 3; i++) {
      final phase = (progress + i / 3) % 1.0;
      final radius = 110 + phase * 50;
      final opacity = (1 - phase) * 0.4;
      paint.color = const Color(0xFF6C63FF).withOpacity(opacity);
      canvas.drawCircle(center, radius, paint);
    }
  }

  @override
  bool shouldRepaint(_WaveRingPainter old) => old.progress != progress;
}

/// Utility to make the custom sine Tweenable — helps wave animation feel natural.
extension _SinExtension on double {
  double sinWave() => (math.sin(this * 2 * math.pi) + 1) / 2;
}
