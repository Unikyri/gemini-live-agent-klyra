import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:klyra/main.dart';

void main() {
  testWidgets('Klyra app boots', (WidgetTester tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: KlyraApp(),
      ),
    );

    expect(find.byType(KlyraApp), findsOneWidget);
  });
}
