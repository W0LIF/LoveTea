import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/providers.dart';

class DatesScreen extends ConsumerWidget {
  const DatesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Идеи свиданий')),
      body: FutureBuilder(
        future: ref.read(datesApiProvider).listIdeas(),
        builder: (context, snapshot) {
          if (snapshot.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(child: Text('Ошибка: ${snapshot.error}'));
          }
          final items = snapshot.data!;
          return ListView.builder(
            itemCount: items.length,
            itemBuilder: (context, index) {
              final item = items[index];
              return ListTile(
                title: Text(item.title),
                subtitle: Text(
                  '${item.category} • ${item.status}',
                ),
                trailing: Text(item.createdAt.toString(),
                    style: const TextStyle(fontSize: 12)),
              );
            },
          );
        },
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () async {
          await ref.read(datesApiProvider).createIdea(
                title: 'Новая идея',
                description: 'Описание',
                category: 'cozy',
              );
        },
        child: const Icon(Icons.add),
      ),
    );
  }
}
