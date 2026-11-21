import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/providers.dart';

class LettersScreen extends ConsumerWidget {
  const LettersScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Письма')),
      body: FutureBuilder(
        future: ref.read(lettersApiProvider).listMyLetters(),
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
                title: Text(item.content),
                subtitle: Text(item.status.toUpperCase()),
                trailing: Text(
                  item.createdAt.toString(),
                  style: const TextStyle(fontSize: 12),
                ),
              );
            },
          );
        },
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () async {
          await ref.read(lettersApiProvider).createLetter(
                recipientUserId: 'partner',
                coupleId: 'couple',
                content: 'Новое письмо',
              );
        },
        child: const Icon(Icons.edit),
      ),
    );
  }
}
