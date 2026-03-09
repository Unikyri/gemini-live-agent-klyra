import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import '../utils/platform_image_url.dart';

/// Unified avatar image widget with platform-aware URL resolution.
/// 
/// Handles avatar display with consistent behavior across screens:
/// - Automatic platform-specific URL resolution (Android emulator support)
/// - Status-aware rendering (pending/generating/ready/failed)
/// - Consistent error fallback (person icon)
/// - Caching for performance
class AvatarImage extends StatelessWidget {
  /// The avatar URL (may be localhost-based, will be resolved for platform)
  final String? avatarUrl;
  
  /// Avatar generation status (pending, generating, ready, failed)
  final String? status;
  
  /// Size of the avatar (width/height)
  final double size;
  
  /// How the image should fit within its bounds
  final BoxFit fit;

  const AvatarImage({
    required this.avatarUrl,
    required this.status,
    this.size = 120,
    this.fit = BoxFit.cover,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    // Show fallback icon if no URL or status not ready
    if (avatarUrl == null || avatarUrl!.isEmpty || status != 'ready') {
      return Icon(
        Icons.person_rounded,
        size: size,
        color: Colors.grey[400],
      );
    }

    // Resolve URL for platform (Android needs 10.0.2.2)
    final resolvedUrl = PlatformImageUrl.resolve(avatarUrl!);

    return CachedNetworkImage(
      imageUrl: resolvedUrl,
      height: size,
      width: size,
      fit: fit,
      placeholder: (context, url) => SizedBox(
        height: size,
        width: size,
        child: const Center(
          child: CircularProgressIndicator(),
        ),
      ),
      errorWidget: (context, url, error) => Icon(
        Icons.person_rounded,
        size: size,
        color: Colors.grey[400],
      ),
    );
  }
}
