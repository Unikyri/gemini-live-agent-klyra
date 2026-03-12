import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';

class CameraSnapshotButton extends StatefulWidget {
  final Future<void> Function(String base64Jpeg) onCaptured;
  final bool enabled;

  const CameraSnapshotButton({
    super.key,
    required this.onCaptured,
    required this.enabled,
  });

  @override
  State<CameraSnapshotButton> createState() => _CameraSnapshotButtonState();
}

class _CameraSnapshotButtonState extends State<CameraSnapshotButton> {
  bool _busy = false;

  @override
  Widget build(BuildContext context) {
    return IconButton(
      tooltip: 'Snapshot',
      onPressed: (!widget.enabled || _busy) ? null : () => _takeSnapshot(context),
      icon: _busy
          ? const SizedBox(
              width: 20,
              height: 20,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : const Icon(Icons.photo_camera_outlined),
    );
  }

  Future<void> _takeSnapshot(BuildContext context) async {
    setState(() => _busy = true);
    try {
      final picker = ImagePicker();
      final file = await picker.pickImage(
        source: ImageSource.camera,
        preferredCameraDevice: CameraDevice.rear,
        imageQuality: 80,
        maxWidth: 1024,
        maxHeight: 1024,
      );
      if (file == null) return;
      final bytes = await file.readAsBytes();
      final b64 = base64Encode(bytes);
      await widget.onCaptured(b64);
    } catch (e) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('No se pudo capturar la foto: $e')),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}

