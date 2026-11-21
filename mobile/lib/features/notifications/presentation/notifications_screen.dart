import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/providers.dart';

class NotificationsScreen extends ConsumerWidget {
  const NotificationsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Уведомления'),
        actions: [
          IconButton(
            icon: const Icon(Icons.mark_email_read),
            onPressed: () async {
              await ref.read(notificationsApiProvider).markAllRead();
            },
          ),
        ],
      ),
      body: FutureBuilder(
        future: ref.read(notificationsApiProvider).list(),
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
                leading: const Icon(Icons.notifications),
                title: Text(item.title),
                subtitle: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(item.body),
                    const SizedBox(height: 4),
                    Text(item.category,
                        style: TextStyle(
                          fontSize: 12,
                          color: Colors.grey[600],
                        )),
                  ],
                ),
                trailing: Icon(
                  item.isRead ? Icons.check_circle : Icons.circle,
                  color: item.isRead ? Colors.green : Colors.blue,
                ),
                onTap: () async {
                  await ref.read(notificationsApiProvider).markRead(item.id);
                },
              );
            },
          );
        },
      ),
    );
  }
}
