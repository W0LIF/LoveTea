import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/network/api_error_message.dart';
import '../../core/providers.dart';

class CoupleJoinScreen extends ConsumerStatefulWidget {
  const CoupleJoinScreen({super.key});

  @override
  ConsumerState<CoupleJoinScreen> createState() => _CoupleJoinScreenState();
}

class _CoupleJoinScreenState extends ConsumerState<CoupleJoinScreen> {
  final _codeController = TextEditingController();
  final _formKey = GlobalKey<FormState>();
  bool _isSubmitting = false;

  @override
  void dispose() {
    _codeController.dispose();
    super.dispose();
  }

  Future<void> _joinByCode() async {
    if (!_formKey.currentState!.validate()) return;

    FocusScope.of(context).unfocus();
    setState(() => _isSubmitting = true);
    try {
      await ref.read(coupleApiProvider).joinByCode(_codeController.text);
      if (mounted) context.go('/app');
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

  Future<void> _pasteCode() async {
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    final pasted = data?.text?.replaceAll(RegExp(r'\D'), '') ?? '';

    if (!mounted) return;

    _codeController.text = pasted.length > 6 ? pasted.substring(0, 6) : pasted;
  }

  @override
  Widget build(BuildContext context) {
    final primary = Theme.of(context).colorScheme.primary;

    return Scaffold(
      appBar: AppBar(title: const Text('Присоединиться'), centerTitle: true),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Form(
            key: _formKey,
            child: Column(
              children: [
                const SizedBox(height: 28),

                Container(
                  width: 100,
                  height: 100,
                  decoration: BoxDecoration(
                    color: const Color(0xFFEAF8F1),
                    borderRadius: BorderRadius.circular(30),
                  ),
                  child: const Icon(
                    Icons.people_alt_rounded,
                    size: 52,
                    color: Color(0xFF71AD8E),
                  ),
                ),

                const SizedBox(height: 24),

                const Text(
                  'Тебя уже ждут!',
                  textAlign: TextAlign.center,
                  style: TextStyle(fontSize: 26, fontWeight: FontWeight.w800),
                ),

                const SizedBox(height: 10),

                Text(
                  'Введи код, который отправил тебе партнёр, '
                  'чтобы присоединиться к вашему общему миру.',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 14,
                    height: 1.5,
                    color: Colors.grey.shade600,
                  ),
                ),

                const SizedBox(height: 36),

                Align(
                  alignment: Alignment.centerLeft,
                  child: Text(
                    'Шестизначный код',
                    style: TextStyle(
                      fontWeight: FontWeight.w700,
                      color: Colors.grey.shade800,
                    ),
                  ),
                ),

                const SizedBox(height: 12),

                TextFormField(
                  controller: _codeController,
                  keyboardType: TextInputType.number,
                  maxLength: 6,
                  textAlign: TextAlign.center,
                  style: const TextStyle(
                    fontSize: 27,
                    fontWeight: FontWeight.w800,
                    letterSpacing: 10,
                  ),
                  decoration: InputDecoration(
                    hintText: '000000',
                    counterText: '',
                    filled: true,
                    fillColor: const Color(0xFFF8F5FF),
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(18),
                      borderSide: BorderSide.none,
                    ),
                    enabledBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(18),
                      borderSide: const BorderSide(color: Color(0xFFE9E0FA)),
                    ),
                    focusedBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(18),
                      borderSide: BorderSide(color: primary, width: 1.5),
                    ),
                  ),
                  validator: (value) {
                    if (value == null || value.length != 6) {
                      return 'Введи все 6 цифр';
                    }

                    if (!RegExp(r'^\d{6}$').hasMatch(value)) {
                      return 'Код должен состоять только из цифр';
                    }

                    return null;
                  },
                ),

                Align(
                  alignment: Alignment.centerRight,
                  child: TextButton.icon(
                    onPressed: _pasteCode,
                    icon: const Icon(Icons.content_paste_rounded),
                    label: const Text('Вставить код'),
                  ),
                ),

                OutlinedButton.icon(
                  onPressed: () => context.push('/couple-scan'),
                  icon: const Icon(Icons.qr_code_scanner_rounded),
                  label: const Text('Сканировать QR-код'),
                ),

                const SizedBox(height: 24),

                SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: _isSubmitting ? null : _joinByCode,
                    style: FilledButton.styleFrom(
                      backgroundColor: primary,
                      padding: const EdgeInsets.symmetric(vertical: 17),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(16),
                      ),
                    ),
                    child: _isSubmitting
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.white,
                            ),
                          )
                        : const Text(
                            'Присоединиться к паре',
                            style: TextStyle(
                              fontSize: 15,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                  ),
                ),

                const SizedBox(height: 28),

                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(18),
                  decoration: BoxDecoration(
                    color: const Color(0xFFF0FAF5),
                    borderRadius: BorderRadius.circular(18),
                  ),
                  child: const Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Icon(
                        Icons.lock_outline_rounded,
                        color: Color(0xFF5B9D7B),
                      ),
                      SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          'Присоединение возможно только по приглашению. '
                          'Оба участника имеют одинаковые права '
                          'и могут управлять настройками своей пары.',
                          style: TextStyle(fontSize: 13, height: 1.45),
                        ),
                      ),
                    ],
                  ),
                ),

                const SizedBox(height: 24),

                TextButton(
                  onPressed: () => Navigator.pop(context),
                  child: const Text('Вернуться назад'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
