import 'dart:ui';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:klyra/features/course/presentation/course_controller.dart';

class CreateCourseModal extends ConsumerStatefulWidget {
  const CreateCourseModal({super.key});

  /// Helper method to show this modal cleanly from anywhere
  static void show(BuildContext context) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (_) => const CreateCourseModal(),
    );
  }

  @override
  ConsumerState<CreateCourseModal> createState() => _CreateCourseModalState();
}

class _CreateCourseModalState extends ConsumerState<CreateCourseModal> {
  final _formKey = GlobalKey<FormState>();
  String _name = '';
  String _educationLevel = 'university';

  final List<String> _levels = ['primary', 'highschool', 'university', 'professional'];

  void _submit() async {
    if (!_formKey.currentState!.validate()) return;
    _formKey.currentState!.save();

    await ref.read(courseControllerProvider.notifier).createCourse(_name, _educationLevel);
    
    if (mounted) {
      // Only close the modal if the state is NOT in error (i.e., creation succeeded)
      final state = ref.read(courseControllerProvider);
      if (!state.hasError) {
        context.pop();
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Failed to create course. Please try again.'),
            backgroundColor: Colors.redAccent,
          ),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isCreating = ref.watch(courseControllerProvider).isLoading;

    return Padding(
      // Push the modal up when the keyboard appears
      padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
      child: ClipRRect(
        borderRadius: const BorderRadius.vertical(top: Radius.circular(24)),
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: 20, sigmaY: 20),
          child: Container(
            padding: const EdgeInsets.all(24),
            decoration: BoxDecoration(
              color: theme.colorScheme.surface.withOpacity(0.8),
              border: Border(top: BorderSide(color: Colors.white.withOpacity(0.1))),
            ),
            child: Form(
              key: _formKey,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Center(
                    child: Container(
                      width: 40,
                      height: 4,
                      margin: const EdgeInsets.only(bottom: 24),
                      decoration: BoxDecoration(
                        color: Colors.white.withOpacity(0.3),
                        borderRadius: BorderRadius.circular(2),
                      ),
                    ),
                  ),
                  Text('New Learning Instance', style: theme.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold)),
                  const SizedBox(height: 24),
                  
                  // Name Field
                  TextFormField(
                    decoration: InputDecoration(
                      labelText: 'Course Name',
                      hintText: 'e.g. Quantum Physics 101',
                      filled: true,
                      fillColor: Colors.black12,
                      border: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: BorderSide.none),
                    ),
                    validator: (val) => val == null || val.trim().isEmpty ? 'Please enter a name' : null,
                    onSaved: (val) => _name = val!.trim(),
                  ),
                  const SizedBox(height: 16),
                  
                  // Education Level Dropdown
                  DropdownButtonFormField<String>(
                    decoration: InputDecoration(
                      labelText: 'Education Level',
                      filled: true,
                      fillColor: Colors.black12,
                      border: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: BorderSide.none),
                    ),
                    value: _educationLevel,
                    items: _levels.map((level) {
                      return DropdownMenuItem(
                        value: level,
                        child: Text(level[0].toUpperCase() + level.substring(1)),
                      );
                    }).toList(),
                    onChanged: (val) {
                      if (val != null) setState(() => _educationLevel = val);
                    },
                    onSaved: (val) => _educationLevel = val!,
                  ),
                  const SizedBox(height: 32),
                  
                  // Submit Button
                  ElevatedButton(
                    onPressed: isCreating ? null : _submit,
                    child: isCreating 
                        ? const SizedBox(width: 24, height: 24, child: CircularProgressIndicator(strokeWidth: 2))
                        : const Text('Create Course', style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
                  ),
                  const SizedBox(height: 16),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
