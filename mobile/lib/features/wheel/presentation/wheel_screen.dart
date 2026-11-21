import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/providers.dart';

class WheelScreen extends ConsumerWidget {
  const WheelScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Колесо')),
      body: Column(
        children: [
          Expanded(
            flex: 2,
            child: FutureBuilder(
              future: ref.read(wheelApiProvider).listSectors(),
              builder: (context, snapshot) {
                if (snapshot.connectionState == ConnectionState.waiting) {
                  return const Center(child: CircularProgressIndicator());
                }
                if (snapshot.hasError) {
                  return Center(child: Text('Ошибка: ${snapshot.error}'));
                }
                final items = snapshot.data!;
                return GridView.builder(
                  gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                    crossAxisCount: 3,
                  ),
                  itemCount: items.length,
                  itemBuilder: (context, index) {
                    final item = items[index];
                    return Card(
                      child: Center(child: Text(item.title)),
                    );
                  },
                );
              },
            ),
          ),
          Expanded(
            flex: 1,
            child: FutureBuilder(
              future: ref.read(wheelApiProvider).history(),
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
                      title: Text('Спин #${index + 1}'),
                      subtitle: Text(item.confirmedByBoth
                          ? 'Подтверждено обоими'
                          : 'Ожидает подтверждения'),
                    );
                  },
                );
              },
            ),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () async {
          await ref.read(wheelApiProvider).spin();
        },
        child: const Icon(Icons.casino),
      ),
    );
  }
}
