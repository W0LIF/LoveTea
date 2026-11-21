import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/providers.dart';

class PetScreen extends ConsumerWidget {
  const PetScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Питомец')),
      body: FutureBuilder(
        future: ref.read(petApiProvider).getPetState(),
        builder: (context, snapshot) {
          if (snapshot.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(child: Text('Ошибка: ${snapshot.error}'));
          }
          final pet = snapshot.data!;
          return Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const Text('Ваш совместный питомец'),
                const SizedBox(height: 24),
                Text('Версия: ${pet.version}'),
                const SizedBox(height: 16),
                Text('Внешность: ${pet.appearance.toString()}'),
                const SizedBox(height: 24),
                ElevatedButton(
                  onPressed: () async {
                    await ref.read(petApiProvider).updateAppearance({
                      'fur_color': 'lavender',
                      'accessory': 'bow',
                      'mood': 'happy',
                    });
                  },
                  child: const Text('Изменить внешность'),
                ),
              ],
            ),
          );
        },
      ),
    );
  }
}
