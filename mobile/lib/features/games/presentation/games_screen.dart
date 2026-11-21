import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/providers.dart';

class GamesScreen extends ConsumerWidget {
  const GamesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Игры')),
      body: FutureBuilder(
        future: ref.read(gamesApiProvider).listSessions(),
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
                leading: const Icon(Icons.games),
                title: Text(item.type.toUpperCase()),
                subtitle: Text(
                  'Счёт: ${item.score} / ${item.totalQuestions}',
                ),
                trailing: Text(item.status.toUpperCase()),
                onTap: () {},
              );
            },
          );
        },
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () async {
          await ref.read(gamesApiProvider).startSession(
                opponentUserId: 'partner',
                type: 'quiz',
              );
        },
        child: const Icon(Icons.add),
      ),
    );
  }
}
