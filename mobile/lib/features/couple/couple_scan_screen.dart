import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../../core/network/api_error_message.dart';
import '../../core/providers.dart';

class CoupleScanScreen extends ConsumerStatefulWidget {
  const CoupleScanScreen({super.key});

  @override
  ConsumerState<CoupleScanScreen> createState() => _CoupleScanScreenState();
}

class _CoupleScanScreenState extends ConsumerState<CoupleScanScreen> {
  final MobileScannerController _controller = MobileScannerController();
  bool _isJoining = false;
  bool _hasScanned = false;

  @override
  void dispose() {
    unawaited(_controller.dispose());
    super.dispose();
  }

  Future<void> _join(String rawValue) async {
    if (_isJoining || _hasScanned) return;

    setState(() {
      _isJoining = true;
      _hasScanned = true;
    });

    try {
      await _controller.stop();
      await ref.read(coupleApiProvider).joinByInvitation(rawValue);
      if (mounted) context.go('/app');
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    } finally {
      if (mounted) setState(() => _isJoining = false);
    }
  }

  Future<void> _scanAgain() async {
    setState(() => _hasScanned = false);
    try {
      await _controller.start();
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    }
  }

  void _onDetect(BarcodeCapture capture) {
    for (final barcode in capture.barcodes) {
      final value = barcode.rawValue?.trim();
      if (value != null && value.isNotEmpty) {
        unawaited(_join(value));
        return;
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Сканировать приглашение'),
        centerTitle: true,
      ),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            children: [
              const Text(
                'Наведи камеру на QR-код приглашения',
                textAlign: TextAlign.center,
                style: TextStyle(fontSize: 17, fontWeight: FontWeight.w700),
              ),
              const SizedBox(height: 20),
              Expanded(
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(24),
                  child: MobileScanner(
                    controller: _controller,
                    onDetect: _onDetect,
                    errorBuilder: (context, error) => ColoredBox(
                      color: Colors.black87,
                      child: Center(
                        child: Padding(
                          padding: const EdgeInsets.all(24),
                          child: Text(
                            'Не удалось открыть камеру. Проверь разрешение '
                            'на использование камеры и попробуй ещё раз.\n'
                            '${error.errorCode.message}',
                            textAlign: TextAlign.center,
                            style: const TextStyle(color: Colors.white),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
              const SizedBox(height: 16),
              if (_isJoining)
                const Padding(
                  padding: EdgeInsets.all(12),
                  child: CircularProgressIndicator(),
                )
              else if (_hasScanned)
                FilledButton.icon(
                  onPressed: _scanAgain,
                  icon: const Icon(Icons.refresh_rounded),
                  label: const Text('Сканировать ещё раз'),
                )
              else
                Text(
                  'Приглашение действует ограниченное время и может быть '
                  'использовано только один раз.',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    color: Colors.grey.shade600,
                    fontSize: 13,
                    height: 1.4,
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}
