import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/providers.dart';

class MomentPhoto extends ConsumerStatefulWidget {
  final String mediaId;

  const MomentPhoto({super.key, required this.mediaId});

  @override
  ConsumerState<MomentPhoto> createState() => _MomentPhotoState();
}

class _MomentPhotoState extends ConsumerState<MomentPhoto> {
  String? _url;
  Object? _error;
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _loadUrl();
  }

  Future<void> _loadUrl() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });
    try {
      final url = await ref
          .read(mediaApiProvider)
          .createDownloadUrl(widget.mediaId);
      if (mounted) setState(() => _url = url);
    } catch (error) {
      if (mounted) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_isLoading) {
      return const SizedBox(
        width: 220,
        height: 170,
        child: Center(child: CircularProgressIndicator()),
      );
    }
    if (_error != null || _url == null) {
      return SizedBox(
        width: 220,
        height: 170,
        child: Center(
          child: TextButton.icon(
            onPressed: _loadUrl,
            icon: const Icon(Icons.refresh_rounded),
            label: const Text('Фото недоступно · повторить'),
          ),
        ),
      );
    }

    return ClipRRect(
      borderRadius: BorderRadius.circular(14),
      child: Image.network(
        _url!,
        width: 220,
        height: 170,
        fit: BoxFit.cover,
        errorBuilder: (context, error, stackTrace) => SizedBox(
          width: 220,
          height: 170,
          child: Center(
            child: TextButton.icon(
              onPressed: _loadUrl,
              icon: const Icon(Icons.refresh_rounded),
              label: const Text('Фото недоступно · повторить'),
            ),
          ),
        ),
      ),
    );
  }
}
