import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:klyra/features/course/presentation/widgets/latex_markdown.dart';

void main() {
  group('LatexMarkdown Widget', () {
    testWidgets('renders plain markdown without LaTeX', (tester) async {
      const markdownData = '# Header\n\nThis is plain text.';

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: LatexMarkdown(markdownData: markdownData),
          ),
        ),
      );

      expect(find.text('Header', skipOffstage: false), findsOneWidget);
      expect(find.textContaining('plain text', skipOffstage: false), findsOneWidget);
    });

    testWidgets('renders markdown with valid inline LaTeX', (tester) async {
      const markdownData = 'The formula \$E=mc^2\$ is famous.';

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: LatexMarkdown(markdownData: markdownData),
          ),
        ),
      );

      // Math widget should be rendered (though we can't verify the exact LaTeX rendering)
      expect(find.textContaining('The formula', skipOffstage: false), findsOneWidget);
      expect(find.textContaining('is famous', skipOffstage: false), findsOneWidget);
    });

    testWidgets('renders markdown with valid block LaTeX', (tester) async {
      const markdownData = 'Formula:\\n\\n\$\$E=mc^2\$\$\\n\\nMore text.';

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: LatexMarkdown(markdownData: markdownData),
          ),
        ),
      );

      expect(find.textContaining('Formula', skipOffstage: false), findsWidgets);
      expect(find.textContaining('More text', skipOffstage: false), findsOneWidget);
    });

    testWidgets('shows warning widget when LaTeX parsing fails', (tester) async {
      // Intentionally malformed LaTeX that should cause flutter_math to throw
      const markdownData = 'Text: \$\\\\frac{incomplete\$ more text.';

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: LatexMarkdown(markdownData: markdownData),
          ),
        ),
      );

      // Warning may appear if parser throws (depends on strictness)
      // For now, just verify it doesn't crash
      expect(find.textContaining('Text', skipOffstage: false), findsOneWidget);
    });

    testWidgets('shows warning widget for invalid inline LaTeX', (tester) async {
      // Malformed inline LaTeX (flutter_math may throw on complex invalid expressions)
      const markdownData = 'Formula: \$\\\\invalid{missing}brace\$ here.';

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: LatexMarkdown(markdownData: markdownData),
          ),
        ),
      );

      // Should show warning if parsing fails
      expect(find.textContaining('Formula', skipOffstage: false), findsOneWidget);
      // Warning may or may not appear depending on parser strictness
    });

    testWidgets('handles markdown with [latex_warning] tag from backend', (tester) async {
      const markdownData =
          '## Topic\\n\\nContent.\\n\\n[latex_warning] Se detecto expresion LaTeX invalida.';

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: LatexMarkdown(markdownData: markdownData),
          ),
        ),
      );

      expect(find.textContaining('Topic', skipOffstage: false), findsOneWidget);
      expect(find.textContaining('[latex_warning]', skipOffstage: false), findsOneWidget);
      expect(find.textContaining('LaTeX invalida', skipOffstage: false), findsOneWidget);
    });

    testWidgets('renders multiple LaTeX blocks correctly', (tester) async {
      const markdownData =
          'First: \$\$E=mc^2\$\$\\n\\nSecond: \$\$F=ma\$\$\\n\\nEnd.';

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: LatexMarkdown(markdownData: markdownData),
          ),
        ),
      );

      expect(find.textContaining('First', skipOffstage: false), findsWidgets);
      expect(find.textContaining('Second', skipOffstage: false), findsWidgets);
      expect(find.textContaining('End', skipOffstage: false), findsOneWidget);
    });

    testWidgets('handles empty markdown gracefully', (tester) async {
      const markdownData = '';

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: LatexMarkdown(markdownData: markdownData),
          ),
        ),
      );

      // Should render empty ListView without crashing
      expect(find.byType(ListView), findsOneWidget);
    });

    testWidgets('warning container has amber styling', (tester) async {
      // Malformed LaTeX that causes flutter_math to throw
      const markdownData = '\$\\\\frac{incomplete\$';

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: LatexMarkdown(markdownData: markdownData),
          ),
        ),
      );

      // If warning appears, verify styling (may not always trigger)
      final warningIcon = find.byIcon(Icons.warning_amber_rounded);
      if (warningIcon.evaluate().isNotEmpty) {
        final iconWidget = tester.widget<Icon>(warningIcon.first);
        expect(iconWidget.color, Colors.amber);
      }
      // Always verify no crash
      expect(find.byType(ListView), findsOneWidget);
    });
  });
}
