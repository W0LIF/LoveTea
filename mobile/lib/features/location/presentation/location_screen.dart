import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/providers.dart';

class LocationScreen extends ConsumerWidget {
  const LocationScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Геолокация')),
      body: FutureBuilder(
        future: ref.read(locationApiProvider).getSharing(),
        builder: (context, snapshot) {
          if (snapshot.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(child: Text('Ошибка: ${snapshot.error}'));
          }
          final item = snapshot.data!;
          return Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const Text('Обмен локацией'),
                const SizedBox(height: 24),
                SwitchListTile(
                  title: const Text('Включить передачу'),
                  value: item.isActive,
                  onChanged: (val) async {
                    await ref
                        .read(locationApiProvider)
                        .toggleSharing(active: val);
                  },
                ),
                const SizedBox(height: 24),
                ElevatedButton(
                  onPressed: () async {
                    await ref.read(locationApiProvider).updatePosition(
                          latitude: 55.7558,
                          longitude: 37.6173,
                        );
                  },
                  child: const Text('Отправить позицию'),
                ),
              ],
            ),
          );
        },
      ),
    );
  }
}
