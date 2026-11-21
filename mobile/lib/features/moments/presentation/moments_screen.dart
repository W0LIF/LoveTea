import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_error_message.dart';
import '../../../core/providers.dart';
import '../../../core/theme/app_theme.dart';
import 'moment_editor_dialog.dart';
import 'moment_photo.dart';

class MomentsScreen extends ConsumerStatefulWidget {
  const MomentsScreen({super.key});

  @override
  ConsumerState<MomentsScreen> createState() => _MomentsScreenState();
}

class _MomentsScreenState extends ConsumerState<MomentsScreen> {
  List<Map<String, dynamic>> _moments = [];
  Object? _error;
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  Future<void> _reload() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });
    try {
      final moments = await ref.read(momentsApiProvider).listMoments();
      if (mounted) setState(() => _moments = moments);
    } catch (error) {
      if (mounted) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  Future<void> _createMoment() async {
    final created = await showDialog<bool>(
      context: context,
      barrierDismissible: false,
      builder: (_) => const MomentEditorDialog(),
    );
    if (created == true && mounted) await _reload();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: const Text('Моменты'),
      actions: [
        IconButton(
          onPressed: _createMoment,
          tooltip: 'Добавить момент',
          icon: const Icon(Icons.add_rounded),
        ),
      ],
    ),
    body: RefreshIndicator(
      onRefresh: _reload,
      child: ListView(
        padding: const EdgeInsets.fromLTRB(18, 8, 18, 30),
        children: [
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              gradient: const LinearGradient(
                colors: [Color(0xFFF7E5E3), Color(0xFFEDE5F5)],
              ),
              borderRadius: BorderRadius.circular(22),
            ),
            child: const Text(
              'Сохраняйте маленькие истории, к которым захочется вернуться.',
              style: TextStyle(height: 1.5, fontWeight: FontWeight.w700),
            ),
          ),
          const SizedBox(height: 16),
          if (_error != null)
            Column(
              children: [
                Text(apiErrorMessage(_error!), textAlign: TextAlign.center),
                const SizedBox(height: 10),
                FilledButton(
                  onPressed: _reload,
                  child: const Text('Повторить'),
                ),
              ],
            )
          else if (_isLoading && _moments.isEmpty)
            const Padding(
              padding: EdgeInsets.only(top: 100),
              child: Center(child: CircularProgressIndicator()),
            )
          else if (_moments.isEmpty)
            const _EmptyMoments()
          else
            ..._moments.map(_momentCard),
        ],
      ),
    ),
    floatingActionButton: FloatingActionButton(
      onPressed: _createMoment,
      child: const Icon(Icons.add_rounded),
    ),
  );

  Widget _momentCard(Map<String, dynamic> moment) {
    final happenedAt = DateTime.parse(
      moment['happened_at'] as String,
    ).toLocal();
    final title = moment['title'] as String? ?? 'Момент';
    final caption = moment['caption'] as String? ?? '';
    final media = moment['media'] is List
        ? (moment['media'] as List)
              .whereType<Map>()
              .map((item) => Map<String, dynamic>.from(item))
              .toList()
        : <Map<String, dynamic>>[];

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Container(
                  width: 42,
                  height: 42,
                  decoration: BoxDecoration(
                    color: AppColors.pink.withValues(alpha: 0.45),
                    borderRadius: BorderRadius.circular(14),
                  ),
                  child: const Icon(
                    Icons.auto_awesome_rounded,
                    color: AppColors.text,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        title,
                        style: const TextStyle(fontWeight: FontWeight.w900),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        '${happenedAt.day.toString().padLeft(2, '0')}.'
                        '${happenedAt.month.toString().padLeft(2, '0')}.'
                        '${happenedAt.year} · '
                        '${moment['scope'] == 'couple' ? 'Для пары' : 'Личное'}',
                        style: const TextStyle(
                          color: AppColors.secondaryText,
                          fontSize: 12,
                        ),
                      ),
                      if (caption.isNotEmpty) ...[
                        const SizedBox(height: 10),
                        Text(caption),
                      ],
                    ],
                  ),
                ),
              ],
            ),
            if (media.isNotEmpty) ...[
              const SizedBox(height: 14),
              SizedBox(
                height: 170,
                child: ListView.separated(
                  scrollDirection: Axis.horizontal,
                  itemCount: media.length,
                  separatorBuilder: (_, _) => const SizedBox(width: 8),
                  itemBuilder: (context, index) {
                    final id = media[index]['id'];
                    if (id is! String || id.isEmpty) {
                      return const SizedBox.shrink();
                    }
                    return MomentPhoto(mediaId: id);
                  },
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _EmptyMoments extends StatelessWidget {
  const _EmptyMoments();

  @override
  Widget build(BuildContext context) => const Padding(
    padding: EdgeInsets.only(top: 45),
    child: Column(
      children: [
        Text('💌', style: TextStyle(fontSize: 48)),
        SizedBox(height: 12),
        Text(
          'Первый момент ещё впереди',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
        ),
        SizedBox(height: 6),
        Text(
          'Добавьте заметку о событии, которое хочется запомнить.',
          textAlign: TextAlign.center,
        ),
      ],
    ),
  );
}
