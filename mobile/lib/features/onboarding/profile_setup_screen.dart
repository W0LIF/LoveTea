import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/network/api_error_message.dart';
import '../../core/providers.dart';
import '../../core/theme/app_theme.dart';

class ProfileSetupScreen extends ConsumerStatefulWidget {
  final String character;
  final String characterName;

  const ProfileSetupScreen({
    super.key,
    required this.character,
    required this.characterName,
  });

  @override
  ConsumerState<ProfileSetupScreen> createState() => _ProfileSetupScreenState();
}

class _ProfileSetupScreenState extends ConsumerState<ProfileSetupScreen> {
  final _formKey = GlobalKey<FormState>();
  final _nameController = TextEditingController();

  int _selectedColor = 0;
  bool _isSubmitting = false;

  final List<Color> _colors = const [
    Color(0xFFC9B6E4),
    Color(0xFFB8DCC8),
    Color(0xFFF4C9D4),
    Color(0xFFFFD9A8),
    Color(0xFFAED9E8),
    Color(0xFFE6C6B8),
  ];

  String get _emoji => widget.character == 'kitty' ? '🐱' : '🐺';

  @override
  void initState() {
    super.initState();
    _nameController.text = widget.characterName;
  }

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  Future<void> _continue() async {
    FocusScope.of(context).unfocus();

    if (!_formKey.currentState!.validate()) return;

    setState(() => _isSubmitting = true);
    try {
      await ref
          .read(profileApiProvider)
          .updateDisplayName(_nameController.text);
      if (!mounted) return;
      context.push(
        '/couple-intro',
        extra: {
          'name': _nameController.text.trim(),
          'about': '',
          'character': widget.character,
          'characterName': widget.characterName,
          'color': _colors[_selectedColor].toARGB32(),
        },
      );
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    } finally {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(24, 12, 24, 30),
          child: Form(
            key: _formKey,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    IconButton(
                      onPressed: () => context.pop(),
                      icon: const Icon(Icons.arrow_back_rounded),
                      style: IconButton.styleFrom(
                        backgroundColor: AppColors.white,
                      ),
                    ),
                    const Spacer(),
                    Text(
                      'ШАГ 2 ИЗ 3',
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
                  'Расскажи о себе',
                  style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                    fontWeight: FontWeight.w900,
                  ),
                ),

                const SizedBox(height: 8),

                Text(
                  'Создай свой маленький уголок. '
                  'Его можно будет изменить в любой момент.',
                  style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                    color: AppColors.secondaryText,
                    height: 1.5,
                  ),
                ),

                const SizedBox(height: 28),

                Center(
                  child: Column(
                    children: [
                      Container(
                        width: 132,
                        height: 132,
                        decoration: BoxDecoration(
                          color: _colors[_selectedColor].withValues(
                            alpha: 0.45,
                          ),
                          shape: BoxShape.circle,
                          border: Border.all(
                            color: _colors[_selectedColor],
                            width: 4,
                          ),
                        ),
                        child: Center(
                          child: Text(
                            _emoji,
                            style: const TextStyle(fontSize: 70),
                          ),
                        ),
                      ),
                      const SizedBox(height: 12),
                      Text(
                        _nameController.text.isEmpty
                            ? 'Твой профиль'
                            : _nameController.text,
                        style: const TextStyle(
                          fontSize: 20,
                          fontWeight: FontWeight.w900,
                          color: AppColors.text,
                        ),
                      ),
                      Text(
                        widget.character == 'kitty'
                            ? 'Лавандовый котёнок'
                            : 'Волчонок зелёного чая',
                        style: TextStyle(
                          color: AppColors.secondaryText,
                          fontSize: 13,
                        ),
                      ),
                    ],
                  ),
                ),

                const SizedBox(height: 30),

                const Text(
                  'Твоё имя',
                  style: TextStyle(fontWeight: FontWeight.w800),
                ),

                const SizedBox(height: 9),

                TextFormField(
                  controller: _nameController,
                  textCapitalization: TextCapitalization.words,
                  maxLength: 24,
                  onChanged: (_) => setState(() {}),
                  validator: (value) {
                    if (value == null || value.trim().isEmpty) {
                      return 'Укажи имя';
                    }
                    if (value.trim().length < 2) {
                      return 'Имя должно содержать хотя бы 2 символа';
                    }
                    return null;
                  },
                  decoration: const InputDecoration(
                    hintText: 'Как тебя называть?',
                    prefixIcon: Icon(Icons.person_outline_rounded),
                    counterText: '',
                  ),
                ),

                const SizedBox(height: 20),

                const Text(
                  'Любимый цвет профиля',
                  style: TextStyle(fontWeight: FontWeight.w800),
                ),

                const SizedBox(height: 14),

                Wrap(
                  spacing: 14,
                  runSpacing: 14,
                  children: List.generate(_colors.length, (index) {
                    final selected = _selectedColor == index;

                    return GestureDetector(
                      onTap: () {
                        setState(() => _selectedColor = index);
                      },
                      child: AnimatedContainer(
                        duration: const Duration(milliseconds: 180),
                        width: 43,
                        height: 43,
                        decoration: BoxDecoration(
                          color: _colors[index],
                          shape: BoxShape.circle,
                          border: Border.all(
                            color: selected
                                ? AppColors.text
                                : Colors.transparent,
                            width: 2.5,
                          ),
                        ),
                        child: selected
                            ? const Icon(
                                Icons.check_rounded,
                                color: Colors.white,
                              )
                            : null,
                      ),
                    );
                  }),
                ),

                const SizedBox(height: 32),

                SizedBox(
                  width: double.infinity,
                  height: 56,
                  child: FilledButton(
                    onPressed: _isSubmitting ? null : _continue,
                    style: FilledButton.styleFrom(
                      backgroundColor: AppColors.lavenderDark,
                      foregroundColor: Colors.white,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(18),
                      ),
                    ),
                    child: _isSubmitting
                        ? const SizedBox(
                            width: 22,
                            height: 22,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.white,
                            ),
                          )
                        : const Text(
                            'Продолжить',
                            style: TextStyle(
                              fontSize: 16,
                              fontWeight: FontWeight.w800,
                            ),
                          ),
                  ),
                ),

                const SizedBox(height: 14),

                Center(
                  child: Text(
                    'Твой профиль — твои правила 💜',
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
      ),
    );
  }
}
