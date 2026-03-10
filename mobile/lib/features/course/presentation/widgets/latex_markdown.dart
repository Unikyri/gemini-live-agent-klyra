import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:flutter_math_fork/flutter_math.dart';
import 'package:markdown/markdown.dart' as md;

class LatexMarkdown extends StatelessWidget {
  final String markdownData;

  const LatexMarkdown({super.key, required this.markdownData});

  @override
  Widget build(BuildContext context) {
    final segments = _splitByBlockLatex(markdownData);
    return ListView.separated(
      itemCount: segments.length,
      separatorBuilder: (_, __) => const SizedBox(height: 12),
      itemBuilder: (context, index) {
        final segment = segments[index];
        if (segment.isLatexBlock) {
          try {
            return Math.tex(segment.content, mathStyle: MathStyle.display);
          } catch (_) {
            return _LatexWarning(content: segment.content);
          }
        }

        return MarkdownBody(
          data: segment.content,
          selectable: true,
          inlineSyntaxes: [InlineLatexSyntax()],
          builders: {'latex': InlineLatexBuilder()},
        );
      },
    );
  }
}

class _Segment {
  final String content;
  final bool isLatexBlock;

  const _Segment(this.content, this.isLatexBlock);
}

List<_Segment> _splitByBlockLatex(String input) {
  final result = <_Segment>[];
  final regex = RegExp(r'\$\$(.*?)\$\$', dotAll: true);
  int currentIndex = 0;

  for (final match in regex.allMatches(input)) {
    if (match.start > currentIndex) {
      final text = input.substring(currentIndex, match.start).trim();
      if (text.isNotEmpty) {
        result.add(_Segment(text, false));
      }
    }

    final latex = (match.group(1) ?? '').trim();
    if (latex.isNotEmpty) {
      result.add(_Segment(latex, true));
    }
    currentIndex = match.end;
  }

  if (currentIndex < input.length) {
    final trailing = input.substring(currentIndex).trim();
    if (trailing.isNotEmpty) {
      result.add(_Segment(trailing, false));
    }
  }

  if (result.isEmpty) {
    result.add(_Segment(input, false));
  }

  return result;
}

class InlineLatexSyntax extends md.InlineSyntax {
  InlineLatexSyntax() : super(r'\$(.+?)\$');

  @override
  bool onMatch(md.InlineParser parser, Match match) {
    final content = match.group(1);
    if (content == null || content.trim().isEmpty) {
      return false;
    }

    final element = md.Element.text('latex', content);
    parser.addNode(element);
    return true;
  }
}

class InlineLatexBuilder extends MarkdownElementBuilder {
  @override
  Widget? visitElementAfter(md.Element element, TextStyle? preferredStyle) {
    final equation = element.textContent;
    try {
      return Math.tex(
        equation,
        mathStyle: MathStyle.text,
        textStyle: preferredStyle,
      );
    } catch (_) {
      return _LatexWarning(content: equation);
    }
  }
}

class _LatexWarning extends StatelessWidget {
  final String content;

  const _LatexWarning({required this.content});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: Colors.amber.withOpacity(0.12),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: Colors.amber.withOpacity(0.5)),
      ),
      child: Row(
        children: [
          const Icon(
            Icons.warning_amber_rounded,
            color: Colors.amber,
            size: 18,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              'LaTeX invalido: $content',
              style: theme.textTheme.bodySmall?.copyWith(
                color: Colors.amber.shade700,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
