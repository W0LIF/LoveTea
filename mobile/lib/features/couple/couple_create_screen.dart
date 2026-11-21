import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../../core/network/api_error_message.dart';
import '../../core/providers.dart';
import 'data/couple_api.dart';

class CoupleCreateScreen extends ConsumerStatefulWidget {
  const CoupleCreateScreen({super.key});

  @override
  ConsumerState<CoupleCreateScreen> createState() => _CoupleCreateScreenState();
}

class _CoupleCreateScreenState extends ConsumerState<CoupleCreateScreen> {
  Timer? _timer;
  CoupleInvitation? _invitation;
  int _secondsLeft = 0;
  bool _isLoading = true;
  bool _isChecking = false;

  @override
  void initState() {
    super.initState();
    _createInvitation();
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  String get _formattedTime {
    final minutes = (_secondsLeft ~/ 60).toString().padLeft(2, '0');
    final seconds = (_secondsLeft % 60).toString().padLeft(2, '0');
    return '$minutes:$seconds';
  }

  Future<void> _createInvitation() async {
    try {
      final invitation = await ref.read(coupleApiProvider).createInvitation();
      if (!mounted) return;
      setState(() {
        _invitation = invitation;
        _secondsLeft = invitation.expiresAt
            .difference(DateTime.now())
            .inSeconds
            .clamp(0, 600);
        _isLoading = false;
      });
      _timer = Timer.periodic(const Duration(seconds: 1), (timer) {
        if (_secondsLeft > 0) {
          setState(() => _secondsLeft--);
        } else {
          timer.cancel();
        }
      });
    } catch (error) {
      if (!mounted) return;
      setState(() => _isLoading = false);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
    }
  }

  Future<void> _renewInvitation() async {
    final coupleId = _invitation?.coupleId;
    if (coupleId == null) return;
    setState(() => _isLoading = true);
    try {
      final invitation = await ref
          .read(coupleApiProvider)
          .renewInvitation(coupleId);
      if (!mounted) return;
      _timer?.cancel();
      setState(() {
        _invitation = invitation;
        _secondsLeft = invitation.expiresAt
            .difference(DateTime.now())
            .inSeconds
            .clamp(0, 600);
        _isLoading = false;
      });
      _timer = Timer.periodic(const Duration(seconds: 1), (timer) {
        if (_secondsLeft > 0) {
          setState(() => _secondsLeft--);
        } else {
          timer.cancel();
        }
      });
    } catch (error) {
      if (!mounted) return;
      setState(() => _isLoading = false);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
    }
  }

  Future<void> _copy(String value, String message) async {
    await Clipboard.setData(ClipboardData(text: value));
    if (!mounted) return;
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));
  }

  Future<void> _checkConnection() async {
    setState(() => _isChecking = true);
    try {
      final connected = await ref.read(coupleApiProvider).hasPartner();
      if (!mounted) return;
      if (connected) {
        context.go('/app');
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Пока ждём второго участника пары.')),
        );
      }
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    } finally {
      if (mounted) setState(() => _isChecking = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final primary = Theme.of(context).colorScheme.primary;

    return Scaffold(
      appBar: AppBar(title: const Text('Создание пары'), centerTitle: true),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Column(
            children: [
              const SizedBox(height: 12),
              Container(
                width: 92,
                height: 92,
                decoration: BoxDecoration(
                  color: const Color(0xFFF0EAFE),
                  borderRadius: BorderRadius.circular(28),
                ),
                child: const Icon(
                  Icons.favorite_rounded,
                  size: 48,
                  color: Color(0xFF9B83D5),
                ),
              ),
              const SizedBox(height: 22),
              const Text(
                'Создаём ваш маленький мир',
                textAlign: TextAlign.center,
                style: TextStyle(fontSize: 23, fontWeight: FontWeight.w800),
              ),
              const SizedBox(height: 10),
              Text(
                'Отправь приглашение своему человеку. '
                'Когда он присоединится, вы сможете пользоваться LoVeTea вместе.',
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 14,
                  height: 1.5,
                  color: Colors.grey.shade600,
                ),
              ),
              const SizedBox(height: 28),
              if (_isLoading)
                const Padding(
                  padding: EdgeInsets.all(24),
                  child: CircularProgressIndicator(),
                )
              else if (_invitation != null)
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(22),
                  decoration: BoxDecoration(
                    color: const Color(0xFFF8F5FF),
                    borderRadius: BorderRadius.circular(24),
                    border: Border.all(color: const Color(0xFFE9E0FA)),
                  ),
                  child: Column(
                    children: [
                      const Text(
                        'Твой код приглашения',
                        style: TextStyle(fontSize: 14),
                      ),
                      const SizedBox(height: 12),
                      Text(
                        '${_invitation!.code.substring(0, 3)} '
                        '${_invitation!.code.substring(3)}',
                        style: const TextStyle(
                          fontSize: 34,
                          fontWeight: FontWeight.w800,
                          letterSpacing: 5,
                        ),
                      ),
                      const SizedBox(height: 20),
                      QrImageView(
                        data: _invitation!.token,
                        version: QrVersions.auto,
                        size: 220,
                        backgroundColor: Colors.white,
                        semanticsLabel: 'QR-код приглашения',
                      ),
                      const SizedBox(height: 10),
                      Text(
                        'Отсканируй QR-код в LoVeTea. Он содержит '
                        'одноразовое приглашение.',
                        textAlign: TextAlign.center,
                        style: TextStyle(
                          color: Colors.grey.shade700,
                          fontSize: 13,
                          height: 1.4,
                        ),
                      ),
                      const SizedBox(height: 16),
                      OutlinedButton.icon(
                        onPressed: () => _copy(
                          _invitation!.code,
                          'Код приглашения скопирован',
                        ),
                        icon: const Icon(Icons.copy_rounded),
                        label: const Text('Скопировать код'),
                      ),
                    ],
                  ),
                )
              else
                const Text(
                  'Не удалось создать приглашение. Вернитесь назад и попробуйте снова.',
                  textAlign: TextAlign.center,
                ),
              if (_invitation != null) ...[
                const SizedBox(height: 16),
                SizedBox(
                  width: double.infinity,
                  child: OutlinedButton.icon(
                    onPressed: () => _copy(
                      _invitation!.token,
                      'Токен приглашения скопирован',
                    ),
                    icon: const Icon(Icons.key_rounded),
                    label: const Text('Скопировать токен'),
                    style: OutlinedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 16),
                    ),
                  ),
                ),
                const SizedBox(height: 18),
                Container(
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: const Color(0xFFF0FAF5),
                    borderRadius: BorderRadius.circular(18),
                  ),
                  child: Row(
                    children: [
                      const Icon(
                        Icons.timer_outlined,
                        color: Color(0xFF5B9D7B),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          _secondsLeft > 0
                              ? 'Код действует ещё'
                              : 'Срок действия кода истёк',
                          style: TextStyle(fontWeight: FontWeight.w600),
                        ),
                      ),
                      Text(
                        _formattedTime,
                        style: const TextStyle(
                          fontWeight: FontWeight.w800,
                          fontSize: 16,
                          color: Color(0xFF478665),
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 18),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: _isChecking ? null : _checkConnection,
                    style: FilledButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 17),
                      backgroundColor: primary,
                    ),
                    child: _isChecking
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.white,
                            ),
                          )
                        : const Text('Проверить подключение'),
                  ),
                ),
                if (_secondsLeft == 0) ...[
                  const SizedBox(height: 8),
                  TextButton.icon(
                    onPressed: _isLoading ? null : _renewInvitation,
                    icon: const Icon(Icons.refresh_rounded),
                    label: const Text('Создать новый код'),
                  ),
                ],
              ],
              const SizedBox(height: 18),
              Text(
                'Вы оба будете равноправными участниками пары.',
                textAlign: TextAlign.center,
                style: TextStyle(color: Colors.grey.shade600, fontSize: 12),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
