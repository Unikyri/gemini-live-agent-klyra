import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:klyra/core/widgets/avatar_image.dart';

void main() {
  group('AvatarImage Widget', () {
    testWidgets('shows fallback icon when avatarUrl is null', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: AvatarImage(
              avatarUrl: null,
              status: 'ready',
            ),
          ),
        ),
      );

      expect(find.byIcon(Icons.person_rounded), findsOneWidget);
      expect(find.byType(CachedNetworkImage), findsNothing);
    });

    testWidgets('shows fallback icon when avatarUrl is empty', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: AvatarImage(
              avatarUrl: '',
              status: 'ready',
            ),
          ),
        ),
      );

      expect(find.byIcon(Icons.person_rounded), findsOneWidget);
      expect(find.byType(CachedNetworkImage), findsNothing);
    });

    testWidgets('shows fallback icon when status is pending', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: AvatarImage(
              avatarUrl: 'http://localhost:8080/static/avatar.png',
              status: 'pending',
            ),
          ),
        ),
      );

      expect(find.byIcon(Icons.person_rounded), findsOneWidget);
      expect(find.byType(CachedNetworkImage), findsNothing);
    });

    testWidgets('shows fallback icon when status is generating', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: AvatarImage(
              avatarUrl: 'http://localhost:8080/static/avatar.png',
              status: 'generating',
            ),
          ),
        ),
      );

      expect(find.byIcon(Icons.person_rounded), findsOneWidget);
      expect(find.byType(CachedNetworkImage), findsNothing);
    });

    testWidgets('shows fallback icon when status is failed', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: AvatarImage(
              avatarUrl: 'http://localhost:8080/static/avatar.png',
              status: 'failed',
            ),
          ),
        ),
      );

      expect(find.byIcon(Icons.person_rounded), findsOneWidget);
      expect(find.byType(CachedNetworkImage), findsNothing);
    });

    testWidgets('shows CachedNetworkImage when status is ready and URL exists', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: AvatarImage(
              avatarUrl: 'http://localhost:8080/static/avatar.png',
              status: 'ready',
            ),
          ),
        ),
      );

      expect(find.byType(CachedNetworkImage), findsOneWidget);
      expect(find.byIcon(Icons.person_rounded), findsNothing);
    });

    testWidgets('applies custom size when provided', (tester) async {
      const customSize = 200.0;
      
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: AvatarImage(
              avatarUrl: null,
              status: 'pending',
              size: customSize,
            ),
          ),
        ),
      );

      final icon = tester.widget<Icon>(find.byIcon(Icons.person_rounded));
      expect(icon.size, customSize);
    });

    testWidgets('applies custom BoxFit when provided', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: AvatarImage(
              avatarUrl: 'http://localhost:8080/static/avatar.png',
              status: 'ready',
              fit: BoxFit.contain,
            ),
          ),
        ),
      );

      final cachedImage = tester.widget<CachedNetworkImage>(
        find.byType(CachedNetworkImage),
      );
      expect(cachedImage.fit, BoxFit.contain);
    });

    testWidgets('uses platform-resolved URL', (tester) async {
      // This test verifies that PlatformImageUrl.resolve is called
      // The actual URL transformation depends on the platform
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: AvatarImage(
              avatarUrl: 'http://localhost:8080/static/avatar.png',
              status: 'ready',
            ),
          ),
        ),
      );

      final cachedImage = tester.widget<CachedNetworkImage>(
        find.byType(CachedNetworkImage),
      );
      
      // Verify imageUrl is set (actual transformation tested in platform_image_url_test)
      expect(cachedImage.imageUrl, isNotEmpty);
    });
  });
}
