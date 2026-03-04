import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:klyra/core/router/app_router.dart';
import 'package:klyra/core/theme/app_theme.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(
    const ProviderScope(
      child: KlyraApp(),
    ),
  );
}

class KlyraApp extends ConsumerWidget {
  const KlyraApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(appRouterProvider);

    return MaterialApp.router(
      title: 'Klyra',
      theme: AppTheme.darkTheme, // Premium dark mode by default
      routerConfig: router,
      debugShowCheckedModeBanner: false,
    );
  }
}
