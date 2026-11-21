import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/storage/onboarding_preferences.dart';
import '../../core/theme/app_theme.dart';

class CharacterScreen extends StatefulWidget {
  const CharacterScreen({super.key});

  @override
  State<CharacterScreen> createState() => _CharacterScreenState();
}

class _CharacterScreenState extends State<CharacterScreen> {
  String _selectedCharacter = 'kitty';
  final _nameController = TextEditingController();

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  void _confirmSelection() {
    final rootContext = context;
    final characterName = _nameController.text.trim();

    if (characterName.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Придумай имя для своего персонажа.')),
      );
      return;
    }

    showDialog<void>(
      context: context,
      builder: (dialogContext) {
        return AlertDialog(
          backgroundColor: AppColors.background,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(24),
          ),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                _selectedCharacter == 'kitty' ? '🐱' : '🐺',
                style: const TextStyle(fontSize: 64),
              ),
              const SizedBox(height: 12),
              Text(
                'Знакомься, $characterName!',
                textAlign: TextAlign.center,
                style: const TextStyle(
                  fontSize: 21,
                  fontWeight: FontWeight.w900,
                  color: AppColors.text,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                'Теперь $characterName — твой маленький спутник в LoVeTea.',
                textAlign: TextAlign.center,
                style: const TextStyle(
                  color: AppColors.secondaryText,
                  height: 1.4,
                ),
              ),
              const SizedBox(height: 20),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: () async {
                    try {
                      await OnboardingPreferences().saveMascot(
                        type: _selectedCharacter == 'kitty' ? 'kitten' : 'wolf',
                        nickname: characterName,
                      );
                    } catch (error) {
                      if (rootContext.mounted && dialogContext.mounted) {
                        ScaffoldMessenger.of(rootContext).showSnackBar(
                          const SnackBar(
                            content: Text(
                              'Не удалось сохранить персонажа. Попробуйте ещё раз.',
                            ),
                          ),
                        );
                      }
                      return;
                    }
                    if (!rootContext.mounted || !dialogContext.mounted) return;
                    Navigator.pop(dialogContext);

                    GoRouter.of(rootContext).push(
                      '/profile-setup',
                      extra: {
                        'character': _selectedCharacter,
                        'characterName': characterName,
                      },
                    );
                  },
                  child: const Text('Ура!'),
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(22, 16, 22, 30),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  IconButton(
                    onPressed: () => Navigator.maybePop(context),
                    icon: const Icon(Icons.arrow_back_rounded),
                    style: IconButton.styleFrom(
                      backgroundColor: AppColors.white,
                    ),
                  ),
                  const Spacer(),
                  Text(
                    'ШАГ 1 ИЗ 3',
                    style: TextStyle(
                      color: AppColors.secondaryText,
                      fontWeight: FontWeight.w800,
                      fontSize: 12,
                      letterSpacing: 1.2,
                    ),
                  ),
                ],
              ),

              const SizedBox(height: 25),

              Text(
                'Кто будет с тобой?',
                style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                  fontWeight: FontWeight.w900,
                ),
              ),

              const SizedBox(height: 9),

              Text(
                'Выбери своего маленького спутника. '
                'Он станет частью твоего мира LoVeTea.',
                style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                  color: AppColors.secondaryText,
                  height: 1.5,
                ),
              ),

              const SizedBox(height: 28),

              _CharacterCard(
                title: 'Kitty',
                subtitle: 'Лавандовый котёнок',
                description:
                    'Любит уют, нежность и тёплые объятия. '
                    'Всегда готов подарить немного мурчания.',
                emoji: '🐱',
                background: const Color(0xFFF0E8F7),
                accent: AppColors.lavenderDark,
                selected: _selectedCharacter == 'kitty',
                onTap: () {
                  setState(() => _selectedCharacter = 'kitty');
                },
              ),

              const SizedBox(height: 16),

              _CharacterCard(
                title: 'Wolf',
                subtitle: 'Волчонок зелёного чая',
                description:
                    'Любит приключения, заботу и спокойные вечера. '
                    'Всегда рядом, когда нужен друг.',
                emoji: '🐺',
                background: const Color(0xFFE7F2EA),
                accent: const Color(0xFF72A987),
                selected: _selectedCharacter == 'wolf',
                onTap: () {
                  setState(() => _selectedCharacter = 'wolf');
                },
              ),

              const SizedBox(height: 28),

              const Text(
                'Как назовём твоего героя?',
                style: TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w800,
                  color: AppColors.text,
                ),
              ),

              const SizedBox(height: 10),

              TextField(
                controller: _nameController,
                textCapitalization: TextCapitalization.words,
                decoration: InputDecoration(
                  hintText: _selectedCharacter == 'kitty'
                      ? 'Например, Kitty'
                      : 'Например, Wolf',
                  prefixIcon: const Icon(Icons.edit_rounded),
                ),
              ),

              const SizedBox(height: 26),

              SizedBox(
                width: double.infinity,
                height: 56,
                child: FilledButton(
                  onPressed: _confirmSelection,
                  style: FilledButton.styleFrom(
                    backgroundColor: AppColors.lavenderDark,
                    foregroundColor: Colors.white,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(18),
                    ),
                  ),
                  child: const Text(
                    'Выбрать персонажа',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800),
                  ),
                ),
              ),

              const SizedBox(height: 14),

              Center(
                child: Text(
                  'Не переживай, выбор можно будет изменить позже 💜',
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

class _CharacterCard extends StatelessWidget {
  final String title;
  final String subtitle;
  final String description;
  final String emoji;
  final Color background;
  final Color accent;
  final bool selected;
  final VoidCallback onTap;

  const _CharacterCard({
    required this.title,
    required this.subtitle,
    required this.description,
    required this.emoji,
    required this.background,
    required this.accent,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(24),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 220),
          padding: const EdgeInsets.all(17),
          decoration: BoxDecoration(
            color: background,
            borderRadius: BorderRadius.circular(24),
            border: Border.all(
              color: selected ? accent : Colors.transparent,
              width: selected ? 2.5 : 1,
            ),
          ),
          child: Row(
            children: [
              Container(
                width: 100,
                height: 116,
                decoration: BoxDecoration(
                  color: Colors.white.withValues(alpha: 0.65),
                  borderRadius: BorderRadius.circular(20),
                ),
                child: Center(
                  child: Text(emoji, style: const TextStyle(fontSize: 62)),
                ),
              ),

              const SizedBox(width: 15),

              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: TextStyle(
                        fontSize: 21,
                        fontWeight: FontWeight.w900,
                        color: AppColors.text,
                      ),
                    ),
                    const SizedBox(height: 3),
                    Text(
                      subtitle,
                      style: TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.w800,
                        color: accent,
                      ),
                    ),
                    const SizedBox(height: 9),
                    Text(
                      description,
                      style: TextStyle(
                        fontSize: 12,
                        height: 1.4,
                        color: AppColors.text,
                      ),
                    ),
                  ],
                ),
              ),

              const SizedBox(width: 8),

              Icon(
                selected ? Icons.check_circle_rounded : Icons.circle_outlined,
                color: selected ? accent : AppColors.secondaryText,
                size: 25,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
