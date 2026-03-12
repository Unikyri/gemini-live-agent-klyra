import 'package:flutter/material.dart';

class DynamicBackground extends StatelessWidget {
  final String contextType;
  final bool enabled;

  const DynamicBackground({
    super.key,
    required this.contextType,
    required this.enabled,
  });

  @override
  Widget build(BuildContext context) {
    final type = enabled ? contextType : 'default';
    return AnimatedSwitcher(
      duration: const Duration(milliseconds: 300),
      child: _BackgroundLayer(
        key: ValueKey(type),
        contextType: type,
      ),
    );
  }
}

class _BackgroundLayer extends StatelessWidget {
  final String contextType;

  const _BackgroundLayer({super.key, required this.contextType});

  @override
  Widget build(BuildContext context) {
    // Prefer assets if present in the app bundle, but fall back to gradients so
    // the feature wiring works even without design assets.
    final asset = switch (contextType) {
      'math' => 'assets/backgrounds/bg_math.webp',
      'science' => 'assets/backgrounds/bg_science.webp',
      'history' => 'assets/backgrounds/bg_history.webp',
      _ => 'assets/backgrounds/bg_default.webp',
    };

    return Positioned.fill(
      child: Image.asset(
        asset,
        fit: BoxFit.cover,
        errorBuilder: (_, __, ___) {
          return DecoratedBox(
            decoration: BoxDecoration(
              gradient: _gradientFor(contextType),
            ),
          );
        },
      ),
    );
  }

  Gradient _gradientFor(String type) {
    return switch (type) {
      'math' => const LinearGradient(
          colors: [Color(0xFF0A0A1A), Color(0xFF0E1A3A)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
      'science' => const LinearGradient(
          colors: [Color(0xFF0A0A1A), Color(0xFF0D2A22)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
      'history' => const LinearGradient(
          colors: [Color(0xFF0A0A1A), Color(0xFF2A160D)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
      _ => const LinearGradient(
          colors: [Color(0xFF0A0A1A), Color(0xFF1A1A2E)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
    };
  }
}

