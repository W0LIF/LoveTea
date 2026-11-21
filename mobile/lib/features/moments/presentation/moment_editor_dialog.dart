import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import '../../../core/network/api_error_message.dart';
import '../../../core/providers.dart';
import '../data/media_api.dart';

class MomentEditorDialog extends ConsumerStatefulWidget {
  const MomentEditorDialog({super.key});

  @override
  ConsumerState<MomentEditorDialog> createState() => _MomentEditorDialogState();
}

class _MomentEditorDialogState extends ConsumerState<MomentEditorDialog> {
  final _titleController = TextEditingController();
  final _captionController = TextEditingController();
  final _photos = <_SelectedPhoto>[];
  String _scope = 'personal';
  DateTime _happenedAt = DateTime.now();
  bool _isSaving = false;
  bool _allowProgrammaticPop = false;
  double? _uploadProgress;
  String? _savingMessage;

  @override
  void dispose() {
    _titleController.dispose();
    _captionController.dispose();
    super.dispose();
  }

  Future<void> _selectPhotos() async {
    try {
      final selected = await ImagePicker().pickMultiImage(imageQuality: 85);
      if (!mounted || selected.isEmpty) return;
      if (_photos.length + selected.length > 20) {
        _showMessage('К одному моменту можно прикрепить не больше 20 фото.');
        return;
      }

      final accepted = <_SelectedPhoto>[];
      final rejected = <String>[];
      for (final file in selected) {
        final contentType = _contentType(file);
        if (contentType == null) {
          rejected.add(file.name);
          continue;
        }
        final size = await file.length();
        if (size == 0 || size > maxMediaUploadBytes) {
          rejected.add(file.name);
          continue;
        }
        accepted.add(_SelectedPhoto(file: file, contentType: contentType));
      }
      if (!mounted) return;
      setState(() => _photos.addAll(accepted));
      if (rejected.isNotEmpty) {
        _showMessage(
          'Пропущены файлы не в формате JPEG/PNG или больше 20 МиБ: '
          '${rejected.join(', ')}',
        );
      }
    } catch (error) {
      if (mounted) _showMessage(apiErrorMessage(error));
    }
  }

  String? _contentType(XFile file) {
    final mimeType = file.mimeType?.toLowerCase();
    if (mimeType == 'image/jpeg' || mimeType == 'image/png') return mimeType;
    final lowerName = file.name.toLowerCase();
    if (lowerName.endsWith('.jpg') || lowerName.endsWith('.jpeg')) {
      return 'image/jpeg';
    }
    if (lowerName.endsWith('.png')) return 'image/png';
    return null;
  }

  Future<void> _removePhoto(_SelectedPhoto photo) async {
    final uploaded = photo.uploaded;
    if (uploaded != null) {
      try {
        await ref.read(mediaApiProvider).deleteMedia(uploaded);
      } catch (error) {
        if (mounted) _showMessage(apiErrorMessage(error));
        return;
      }
    }
    if (mounted) setState(() => _photos.remove(photo));
  }

  Future<void> _cancel() async {
    try {
      for (final photo in _photos) {
        final media = photo.uploaded;
        if (media == null) continue;
        await ref.read(mediaApiProvider).deleteMedia(media);
        photo.uploaded = null;
      }
      if (mounted) await _close(false);
    } catch (error) {
      if (mounted) _showMessage(apiErrorMessage(error));
    }
  }

  Future<void> _submit() async {
    final title = _titleController.text.trim();
    final caption = _captionController.text.trim();
    if (title.isEmpty || title.runes.length > 200) {
      _showMessage('Добавьте заголовок длиной до 200 символов.');
      return;
    }
    if (caption.runes.length > 5000) {
      _showMessage('Заметка не должна превышать 5000 символов.');
      return;
    }

    setState(() {
      _isSaving = true;
      _uploadProgress = _photos.isEmpty ? null : 0;
      _savingMessage = _photos.isEmpty
          ? 'Сохраняем момент…'
          : 'Загружаем фото…';
    });

    try {
      final mediaIds = <String>[];
      for (var index = 0; index < _photos.length; index++) {
        final photo = _photos[index];
        var media = photo.uploaded;
        if (media == null) {
          final bytes = await photo.file.readAsBytes();
          media = await ref
              .read(mediaApiProvider)
              .uploadImage(
                bytes: bytes,
                fileName: photo.file.name,
                contentType: photo.contentType,
                scope: _scope,
                onSendProgress: (sent, total) {
                  if (!mounted || total <= 0) return;
                  setState(() {
                    _uploadProgress = (index + sent / total) / _photos.length;
                    _savingMessage =
                        'Загружаем фото ${index + 1} из ${_photos.length}…';
                  });
                },
              );
          photo.uploaded = media;
        }
        mediaIds.add(media.id);
      }

      await ref
          .read(momentsApiProvider)
          .createMoment(
            title: title,
            caption: caption,
            scope: _scope,
            happenedAt: _happenedAt,
            mediaIds: mediaIds,
          );
      if (mounted) await _close(true);
    } catch (error) {
      if (mounted) _showMessage(apiErrorMessage(error));
    } finally {
      if (mounted) {
        setState(() {
          _isSaving = false;
          _uploadProgress = null;
          _savingMessage = null;
        });
      }
    }
  }

  void _showMessage(String message) {
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));
  }

  Future<void> _close(bool result) async {
    setState(() => _allowProgrammaticPop = true);
    await WidgetsBinding.instance.endOfFrame;
    if (mounted) Navigator.pop(context, result);
  }

  @override
  Widget build(BuildContext context) => PopScope(
    canPop:
        _allowProgrammaticPop ||
        (!_isSaving && _photos.every((photo) => photo.uploaded == null)),
    child: AlertDialog(
      title: const Text('Сохранить момент'),
      content: SizedBox(
        width: 480,
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: _titleController,
                autofocus: true,
                maxLength: 200,
                enabled: !_isSaving,
                decoration: const InputDecoration(labelText: 'Заголовок'),
              ),
              const SizedBox(height: 8),
              TextField(
                controller: _captionController,
                maxLength: 5000,
                maxLines: 3,
                enabled: !_isSaving,
                decoration: const InputDecoration(
                  labelText: 'Заметка',
                  alignLabelWithHint: true,
                ),
              ),
              const SizedBox(height: 8),
              DropdownButtonFormField<String>(
                initialValue: _scope,
                decoration: const InputDecoration(labelText: 'Видимость'),
                items: const [
                  DropdownMenuItem(
                    value: 'personal',
                    child: Text('Только для меня'),
                  ),
                  DropdownMenuItem(value: 'couple', child: Text('Для пары')),
                ],
                onChanged: _isSaving
                    ? null
                    : (value) {
                        if (value != null) setState(() => _scope = value);
                      },
              ),
              const SizedBox(height: 8),
              ListTile(
                contentPadding: EdgeInsets.zero,
                leading: const Icon(Icons.calendar_today_rounded),
                title: const Text('Дата момента'),
                subtitle: Text(
                  '${_happenedAt.day.toString().padLeft(2, '0')}.'
                  '${_happenedAt.month.toString().padLeft(2, '0')}.'
                  '${_happenedAt.year}',
                ),
                onTap: _isSaving ? null : _selectDate,
              ),
              Align(
                alignment: Alignment.centerLeft,
                child: OutlinedButton.icon(
                  onPressed: _isSaving ? null : _selectPhotos,
                  icon: const Icon(Icons.add_photo_alternate_outlined),
                  label: Text(
                    _photos.isEmpty
                        ? 'Добавить фото'
                        : 'Добавить ещё фото (${_photos.length}/20)',
                  ),
                ),
              ),
              if (_photos.isNotEmpty)
                Align(
                  alignment: Alignment.centerLeft,
                  child: Wrap(
                    spacing: 6,
                    runSpacing: 4,
                    children: _photos.map((photo) {
                      return InputChip(
                        avatar: const Icon(Icons.image_outlined, size: 18),
                        label: ConstrainedBox(
                          constraints: const BoxConstraints(maxWidth: 180),
                          child: Text(
                            photo.file.name,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        onDeleted: _isSaving ? null : () => _removePhoto(photo),
                      );
                    }).toList(),
                  ),
                ),
              const SizedBox(height: 6),
              const Align(
                alignment: Alignment.centerLeft,
                child: Text(
                  'Поддерживаются JPEG и PNG размером до 20 МиБ на фото.',
                  style: TextStyle(fontSize: 12, color: Color(0xFF918A99)),
                ),
              ),
              if (_isSaving) ...[
                const SizedBox(height: 16),
                LinearProgressIndicator(value: _uploadProgress),
                const SizedBox(height: 8),
                Text(_savingMessage ?? 'Сохраняем…'),
              ],
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: _isSaving ? null : _cancel,
          child: const Text('Отмена'),
        ),
        FilledButton(
          onPressed: _isSaving ? null : _submit,
          child: _isSaving
              ? const SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    color: Colors.white,
                  ),
                )
              : const Text('Сохранить'),
        ),
      ],
    ),
  );

  Future<void> _selectDate() async {
    final selected = await showDatePicker(
      context: context,
      initialDate: _happenedAt,
      firstDate: DateTime(1900),
      lastDate: DateTime.now(),
    );
    if (selected == null || !mounted) return;
    setState(() {
      _happenedAt = DateTime(
        selected.year,
        selected.month,
        selected.day,
        _happenedAt.hour,
        _happenedAt.minute,
      );
    });
  }
}

class _SelectedPhoto {
  final XFile file;
  final String contentType;
  MediaFile? uploaded;

  _SelectedPhoto({required this.file, required this.contentType});
}
