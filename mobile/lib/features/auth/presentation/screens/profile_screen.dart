import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_error_message.dart';
import '../../../../core/providers.dart';
import '../../../../core/theme/app_theme.dart';
import '../../data/profile_api.dart';

class ProfileScreen extends ConsumerStatefulWidget {
  const ProfileScreen({super.key});

  @override
  ConsumerState<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends ConsumerState<ProfileScreen> {
  final _nameController = TextEditingController();
  UserProfile? _profile;
  Object? _error;
  bool _isLoading = true;
  bool _isSaving = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });
    try {
      final profile = await ref.read(profileApiProvider).getProfile();
      if (!mounted) return;
      setState(() {
        _profile = profile;
        _nameController.text = profile.displayName ?? '';
      });
    } catch (error) {
      if (mounted) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  Future<void> _save() async {
    final name = _nameController.text.trim();
    if (name.isEmpty || name.runes.length > 50) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Имя должно содержать от 1 до 50 символов.'),
        ),
      );
      return;
    }

    setState(() => _isSaving = true);
    try {
      await ref.read(profileApiProvider).updateDisplayName(name);
      if (mounted) Navigator.pop(context, true);
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    } finally {
      if (mounted) setState(() => _isSaving = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Профиль')),
    body: _isLoading
        ? const Center(child: CircularProgressIndicator())
        : _error != null
        ? Center(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(apiErrorMessage(_error!), textAlign: TextAlign.center),
                  const SizedBox(height: 12),
                  FilledButton(
                    onPressed: _load,
                    child: const Text('Повторить'),
                  ),
                ],
              ),
            ),
          )
        : ListView(
            padding: const EdgeInsets.all(20),
            children: [
              Container(
                padding: const EdgeInsets.all(20),
                decoration: BoxDecoration(
                  color: const Color(0xFFF2EAF8),
                  borderRadius: BorderRadius.circular(22),
                ),
                child: const Text(
                  'Эти данные используются для приветствия и вашего профиля.',
                  style: TextStyle(height: 1.5, fontWeight: FontWeight.w600),
                ),
              ),
              const SizedBox(height: 22),
              TextField(
                key: const ValueKey('profile-display-name'),
                controller: _nameController,
                maxLength: 50,
                enabled: !_isSaving,
                decoration: const InputDecoration(
                  labelText: 'Как к вам обращаться',
                  prefixIcon: Icon(Icons.person_outline_rounded),
                ),
              ),
              const SizedBox(height: 10),
              InputDecorator(
                decoration: const InputDecoration(
                  labelText: 'Электронная почта',
                  prefixIcon: Icon(Icons.mail_outline_rounded),
                ),
                child: Text(
                  _profile?.email ?? '',
                  style: const TextStyle(color: AppColors.secondaryText),
                ),
              ),
              const SizedBox(height: 22),
              FilledButton(
                onPressed: _isSaving ? null : _save,
                child: _isSaving
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : const Text('Сохранить профиль'),
              ),
            ],
          ),
  );
}
