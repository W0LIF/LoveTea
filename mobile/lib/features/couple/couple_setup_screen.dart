import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/theme/app_theme.dart';

class CoupleSetupScreen extends StatelessWidget {
  const CoupleSetupScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(24, 16, 24, 28),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              IconButton(
                onPressed: () => context.pop(),
                icon: const Icon(Icons.arrow_back_rounded),
                style: IconButton.styleFrom(backgroundColor: AppColors.white),
              ),

              const SizedBox(height: 35),

              Text(
                'Ваш мир начинается вдвоём',
                style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                  fontWeight: FontWeight.w900,
                ),
              ),

              const SizedBox(height: 10),

              Text(
                'Создай своё пространство и пригласи любимого человека '
                'или присоединись к уже созданному.',
                style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                  color: AppColors.secondaryText,
                  height: 1.5,
                ),
              ),

              const SizedBox(height: 35),

              _CoupleOption(
                icon: Icons.favorite_rounded,
                background: const Color(0xFFF1EAF8),
                accent: AppColors.lavenderDark,
                title: 'Создать нашу пару',
                description:
                    'Создай приглашение и поделись им со вторым человеком.',
                buttonText: 'Создать приглашение',
                onTap: () => context.push('/couple-create'),
              ),

              const SizedBox(height: 18),

              _CoupleOption(
                icon: Icons.mail_outline_rounded,
                background: const Color(0xFFE8F3EC),
                accent: const Color(0xFF72A987),
                title: 'У меня есть приглашение',
                description:
                    'Введи код или открой ссылку, которую тебе отправили.',
                buttonText: 'Присоединиться',
                onTap: () => context.push('/couple-join'),
              ),

              const Spacer(),

              Center(
                child: Text(
                  'Вы оба равноправные участники вашего мира 💜',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    color: AppColors.secondaryText,
                    fontSize: 12,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _CoupleOption extends StatelessWidget {
  final IconData icon;
  final Color background;
  final Color accent;
  final String title;
  final String description;
  final String buttonText;
  final VoidCallback onTap;

  const _CoupleOption({
    required this.icon,
    required this.background,
    required this.accent,
    required this.title,
    required this.description,
    required this.buttonText,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(24),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 32, color: accent),
          const SizedBox(height: 15),
          Text(
            title,
            style: const TextStyle(
              fontSize: 19,
              fontWeight: FontWeight.w900,
              color: AppColors.text,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            description,
            style: const TextStyle(
              color: AppColors.secondaryText,
              height: 1.45,
            ),
          ),
          const SizedBox(height: 18),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: onTap,
              style: FilledButton.styleFrom(
                backgroundColor: accent,
                foregroundColor: Colors.white,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(15),
                ),
              ),
              child: Text(buttonText),
            ),
          ),
        ],
      ),
    );
  }
}
