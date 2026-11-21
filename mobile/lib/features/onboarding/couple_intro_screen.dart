import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/theme/app_theme.dart';

class CoupleIntroScreen extends StatelessWidget {
  final String name;
  final String about;
  final String character;
  final String characterName;
  final int color;

  const CoupleIntroScreen({
    super.key,
    required this.name,
    required this.about,
    required this.character,
    required this.characterName,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    final isKitty = character == 'kitty';

    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(26, 20, 26, 28),
          child: Column(
            children: [
              Align(
                alignment: Alignment.centerRight,
                child: Text(
                  'ШАГ 3 ИЗ 3',
                  style: TextStyle(
                    color: AppColors.secondaryText,
                    fontWeight: FontWeight.w800,
                    fontSize: 12,
                    letterSpacing: 1.2,
                  ),
                ),
              ),

              const Spacer(),

              Container(
                width: 210,
                height: 210,
                decoration: BoxDecoration(
                  color: AppColors.lavender.withValues(alpha: 0.25),
                  shape: BoxShape.circle,
                ),
                child: Center(
                  child: Text(
                    isKitty ? '🐱 💜 🐺' : '🐺 💚 🐱',
                    style: const TextStyle(fontSize: 55),
                  ),
                ),
              ),

              const SizedBox(height: 35),

              Text(
                'Добро пожаловать, $name!',
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                  fontWeight: FontWeight.w900,
                ),
              ),

              const SizedBox(height: 14),

              Text(
                'LoVeTea — это ваше личное пространство, '
                'где можно хранить воспоминания, заботиться друг о друге '
                'и создавать маленькие традиции.',
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                  color: AppColors.secondaryText,
                  height: 1.6,
                ),
              ),

              const SizedBox(height: 30),

              _IntroFeature(
                icon: Icons.favorite_rounded,
                color: AppColors.pink,
                title: 'Ваш общий мир',
                description: 'Моменты, фотографии и важные даты',
              ),

              const SizedBox(height: 14),

              _IntroFeature(
                icon: Icons.chat_bubble_rounded,
                color: AppColors.lavender,
                title: 'Больше общения',
                description: 'Чат, желания и маленькие сюрпризы',
              ),

              const SizedBox(height: 14),

              _IntroFeature(
                icon: Icons.pets_rounded,
                color: AppColors.mint,
                title: 'Ваши персонажи',
                description: 'Kitty и Wolf будут частью вашей истории',
              ),

              const Spacer(),

              SizedBox(
                width: double.infinity,
                height: 56,
                child: FilledButton(
                  onPressed: () {
                    context.go('/couple-setup');
                  },
                  style: FilledButton.styleFrom(
                    backgroundColor: AppColors.lavenderDark,
                    foregroundColor: Colors.white,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(18),
                    ),
                  ),
                  child: const Text(
                    'Создать наш мир',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800),
                  ),
                ),
              ),

              const SizedBox(height: 14),

              Text(
                'Всё начинается с маленьких моментов 💜',
                style: TextStyle(color: AppColors.secondaryText, fontSize: 12),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _IntroFeature extends StatelessWidget {
  final IconData icon;
  final Color color;
  final String title;
  final String description;

  const _IntroFeature({
    required this.icon,
    required this.color,
    required this.title,
    required this.description,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 46,
          height: 46,
          decoration: BoxDecoration(
            color: color.withValues(alpha: 0.35),
            borderRadius: BorderRadius.circular(15),
          ),
          child: Icon(icon, color: AppColors.text),
        ),
        const SizedBox(width: 14),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(title, style: const TextStyle(fontWeight: FontWeight.w800)),
              const SizedBox(height: 3),
              Text(
                description,
                style: TextStyle(fontSize: 12, color: AppColors.secondaryText),
              ),
            ],
          ),
        ),
      ],
    );
  }
}
