import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/network/api_error_message.dart';
import '../../core/providers.dart';
import '../../core/theme/app_theme.dart';
import '../games/presentation/games_screen.dart';
import '../habits/presentation/habits_screen.dart';
import '../moments/presentation/moments_screen.dart';
import '../notifications/presentation/notifications_screen.dart';
import '../pet/presentation/pet_screen.dart';
import '../planning/presentation/planning_screen.dart';
import '../wellness/presentation/wellness_screen.dart';
import '../wheel/presentation/wheel_screen.dart';
import 'data/home_api.dart';
import 'data/home_upcoming.dart';

class MainShell extends ConsumerStatefulWidget {
  const MainShell({super.key});

  @override
  ConsumerState<MainShell> createState() => _MainShellState();
}

class _MainShellState extends ConsumerState<MainShell> {
  int _selectedIndex = 0;
  final Set<int> _visitedTabs = {0};

  @override
  Widget build(BuildContext context) {
    final pages = <Widget>[
      HomeTab(
        onLogout: _logout,
        onNavigate: (index) => setState(() => _selectedIndex = index),
      ),
      const PlanningScreen(),
      const HabitsScreen(),
      const WellnessScreen(),
      const MomentsScreen(),
      const PetScreen(),
      const NotificationsScreen(),
      const GamesScreen(),
      const WheelScreen(),
    ];
    final visiblePages = List<Widget>.generate(
      pages.length,
      (index) =>
          _visitedTabs.contains(index) ? pages[index] : const SizedBox.shrink(),
    );

    return Scaffold(
      body: IndexedStack(index: _selectedIndex, children: visiblePages),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _selectedIndex,
        onDestinationSelected: (index) {
          setState(() {
            _selectedIndex = index;
            _visitedTabs.add(index);
          });
        },
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.home_outlined),
            selectedIcon: Icon(Icons.home_rounded),
            label: 'Домой',
          ),
          NavigationDestination(
            icon: Icon(Icons.event_note_outlined),
            selectedIcon: Icon(Icons.event_note_rounded),
            label: 'План',
          ),
          NavigationDestination(
            icon: Icon(Icons.spa_outlined),
            selectedIcon: Icon(Icons.spa_rounded),
            label: 'Привычки',
          ),
          NavigationDestination(
            icon: Icon(Icons.mood_outlined),
            selectedIcon: Icon(Icons.mood_rounded),
            label: 'Настроение',
          ),
          NavigationDestination(
            icon: Icon(Icons.auto_awesome_outlined),
            selectedIcon: Icon(Icons.auto_awesome_rounded),
            label: 'Моменты',
          ),
          NavigationDestination(
            icon: Icon(Icons.pets_outlined),
            selectedIcon: Icon(Icons.pets_rounded),
            label: 'Питомец',
          ),
          NavigationDestination(
            icon: Icon(Icons.notifications_outlined),
            selectedIcon: Icon(Icons.notifications_rounded),
            label: 'Уведомления',
          ),
          NavigationDestination(
            icon: Icon(Icons.games_outlined),
            selectedIcon: Icon(Icons.games_rounded),
            label: 'Игры',
          ),
          NavigationDestination(
            icon: Icon(Icons.casino_outlined),
            selectedIcon: Icon(Icons.casino_rounded),
            label: 'Колесо',
          ),
        ],
      ),
    );
  }

  Future<void> _logout() async {
    try {
      await ref.read(authApiProvider).logout();
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    } finally {
      if (mounted) context.go('/');
    }
  }
}

class HomeTab extends ConsumerStatefulWidget {
  final Future<void> Function() onLogout;
  final ValueChanged<int> onNavigate;

  const HomeTab({super.key, required this.onLogout, required this.onNavigate});

  @override
  ConsumerState<HomeTab> createState() => _HomeTabState();
}

class _HomeTabState extends ConsumerState<HomeTab> {
  HomeSnapshot? _snapshot;
  List<HomeUpcomingItem> _upcomingItems = [];
  Map<String, dynamic>? _latestMoment;
  Object? _error;
  Object? _syncError;
  Object? _upcomingError;
  bool _isLoading = true;
  bool _isSyncing = false;
  bool _isUpcomingLoading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });
    try {
      final snapshot = await ref.read(homeApiProvider).load();
      if (!mounted) return;
      setState(() => _snapshot = snapshot);
      ref.read(currentUserIdProvider.notifier).state = snapshot.userId;
      unawaited(_loadUpcoming(snapshot.userId));
      unawaited(_loadLatestMoment());
      unawaited(_syncChanges(snapshot.userId));
      if (snapshot.coupleStatus == 'active') {
        try {
          await _applySavedMascot();
        } catch (error) {
          if (mounted) {
            ScaffoldMessenger.of(
              context,
            ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
          }
        }
      }
    } catch (error) {
      if (mounted) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  Future<void> _syncChanges(String userId) async {
    setState(() {
      _isSyncing = true;
      _syncError = null;
    });
    try {
      await ref.read(syncServiceProvider).synchronize(userId);
    } catch (error) {
      if (mounted) setState(() => _syncError = error);
    } finally {
      if (mounted) setState(() => _isSyncing = false);
    }
  }

  Future<void> _loadLatestMoment() async {
    try {
      final moments = await ref.read(momentsApiProvider).listMoments();
      moments.sort((first, second) {
        final firstDate = DateTime.parse(first['happened_at'] as String);
        final secondDate = DateTime.parse(second['happened_at'] as String);
        return secondDate.compareTo(firstDate);
      });
      if (mounted) {
        setState(() => _latestMoment = moments.isEmpty ? null : moments.first);
      }
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    }
  }

  Future<void> _loadUpcoming(String userId) async {
    setState(() {
      _isUpcomingLoading = true;
      _upcomingError = null;
    });
    try {
      final api = ref.read(planningApiProvider);
      final results = await Future.wait([
        api.listTasks(userId: userId),
        api.listEvents(userId: userId),
        api.listImportantDates(userId: userId),
      ]);
      if (!mounted) return;
      setState(() {
        _upcomingItems = buildHomeUpcomingItems(
          today: DateTime.now(),
          tasks: results[0],
          events: results[1],
          importantDates: results[2],
        );
      });
    } catch (error) {
      if (mounted) setState(() => _upcomingError = error);
    } finally {
      if (mounted) setState(() => _isUpcomingLoading = false);
    }
  }

  Future<void> _applySavedMascot() async {
    final preferences = ref.read(onboardingPreferencesProvider);
    final mascot = await preferences.getMascot();
    if (mascot == null) return;
    await ref
        .read(coupleApiProvider)
        .updateMemberProfile(
          nickname: mascot.nickname,
          mascotType: mascot.type,
        );
    await preferences.clearMascot();
  }

  Future<void> _requestDetach() async {
    final confirmedByUser = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Отвязать пару?'),
        content: const Text(
          'Это подтверждение будет отправлено серверу. Пара отвяжется только '
          'после подтверждения обоими участниками; одного вашего действия '
          'недостаточно.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, false),
            child: const Text('Оставить как есть'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(dialogContext, true),
            child: const Text('Подтвердить'),
          ),
        ],
      ),
    );
    if (confirmedByUser != true || !mounted) return;

    try {
      final result = await ref.read(coupleApiProvider).confirmDetach();
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            result.confirmed
                ? 'Оба участника подтвердили отвязку пары.'
                : 'Ваше подтверждение сохранено. Ожидается решение партнёра.',
          ),
        ),
      );
      await _load();
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    }
  }

  Future<void> _openProfile() async {
    final updated = await context.push<bool>('/profile');
    if (updated == true && mounted) await _load();
  }

  @override
  Widget build(BuildContext context) {
    final snapshot = _snapshot;

    return Scaffold(
      appBar: AppBar(
        title: const Text('LoVeTea'),
        actions: [
          PopupMenuButton<String>(
            onSelected: (value) {
              if (value == 'logout') widget.onLogout();
              if (value == 'detach') _requestDetach();
              if (value == 'profile') _openProfile();
            },
            itemBuilder: (context) => [
              const PopupMenuItem(value: 'profile', child: Text('Профиль')),
              if (snapshot?.coupleStatus == 'active')
                const PopupMenuItem(
                  value: 'detach',
                  child: Text('Отвязать пару'),
                ),
              const PopupMenuItem(
                value: 'logout',
                child: Text('Выйти из аккаунта'),
              ),
            ],
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(20, 8, 20, 28),
          children: [
            if (_isLoading && snapshot == null)
              const SizedBox(
                height: 360,
                child: Center(child: CircularProgressIndicator()),
              )
            else if (_error != null && snapshot == null)
              _ErrorCard(error: _error!, onRetry: _load)
            else if (snapshot != null) ...[
              _WelcomeCard(snapshot: snapshot),
              const SizedBox(height: 18),
              if (_isSyncing || _syncError != null)
                _SyncStatusCard(
                  isSyncing: _isSyncing,
                  error: _syncError,
                  onRetry: () => _syncChanges(snapshot.userId),
                ),
              if (_isSyncing || _syncError != null) const SizedBox(height: 18),
              if (snapshot.coupleStatus == null)
                _StartCoupleCard()
              else if (snapshot.coupleStatus == 'pending')
                const _PendingCoupleCard()
              else
                _CoupleCard(snapshot: snapshot),
              const SizedBox(height: 18),
              _HomeUpcomingCard(
                items: _upcomingItems,
                isLoading: _isUpcomingLoading,
                error: _upcomingError,
                onRetry: () {
                  final userId = _snapshot?.userId;
                  if (userId != null) _loadUpcoming(userId);
                },
                onTap: () => widget.onNavigate(1),
              ),
              const SizedBox(height: 18),
              if (_latestMoment != null) ...[
                _LatestMomentCard(
                  moment: _latestMoment!,
                  onTap: () => widget.onNavigate(4),
                ),
                const SizedBox(height: 18),
              ],
              if (snapshot.coupleStatus == 'active' &&
                  snapshot.pet != null) ...[
                _PetStateCard(pet: snapshot.pet!),
                const SizedBox(height: 18),
              ],
              const _SectionHeading(title: 'Ваш маленький мир'),
              const SizedBox(height: 10),
              _FeatureCard(
                icon: Icons.event_available_rounded,
                title: 'Планы и важные даты',
                subtitle: 'Держите общие дела и события под рукой',
                color: AppColors.lavender,
                onTap: () => widget.onNavigate(1),
              ),
              const SizedBox(height: 10),
              _FeatureCard(
                icon: Icons.spa_rounded,
                title: 'Привычки пары',
                subtitle: 'Маленькие шаги, которые вы делаете вместе',
                color: AppColors.mint,
                onTap: () => widget.onNavigate(2),
              ),
              const SizedBox(height: 10),
              _FeatureCard(
                icon: Icons.favorite_rounded,
                title: 'Забота о себе',
                subtitle: 'Отмечайте настроение — заметки приватны',
                color: AppColors.pink,
                onTap: () => widget.onNavigate(3),
              ),
              const SizedBox(height: 10),
              _FeatureCard(
                icon: Icons.auto_awesome_rounded,
                title: 'Моменты',
                subtitle: 'Сохраняйте тёплые истории',
                color: Color(0xFFFFD9A8),
                onTap: () => widget.onNavigate(4),
              ),
              const SizedBox(height: 10),
              _FeatureCard(
                icon: Icons.psychology_alt_rounded,
                title: 'Знакомство с собой',
                subtitle: 'Ответьте на вопросы опубликованных тестов',
                color: AppColors.lavender,
                onTap: () => context.push('/onboarding-tests'),
              ),
              const SizedBox(height: 10),
              _FeatureCard(
                icon: Icons.chat_bubble_outline_rounded,
                title: 'Чат',
                subtitle: 'Обмен сообщениями в режиме реального времени',
                color: AppColors.lavender,
                onTap: () => context.push('/chat'),
              ),
              const SizedBox(height: 10),
              _FeatureCard(
                icon: Icons.edit_note_rounded,
                title: 'Письма',
                subtitle: 'Гневное и спокойное письмо с охлаждением',
                color: AppColors.mint,
                onTap: () => context.push('/letters'),
              ),
              const SizedBox(height: 10),
              _FeatureCard(
                icon: Icons.location_on_rounded,
                title: 'Геолокация',
                subtitle: 'Добровольный обмен местоположением',
                color: AppColors.pink,
                onTap: () => context.push('/location'),
              ),
              const SizedBox(height: 10),
              _FeatureCard(
                icon: Icons.calendar_today_rounded,
                title: 'Идеи свиданий',
                subtitle: 'Рекомендации на основе предпочтений пары',
                color: Color(0xFFFFD9A8),
                onTap: () => context.push('/dates'),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _WelcomeCard extends StatelessWidget {
  final HomeSnapshot snapshot;

  const _WelcomeCard({required this.snapshot});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(22),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFFE9DDF5), Color(0xFFF9E5E5)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(26),
      ),
      child: Row(
        children: [
          const Text('🐱 💜 🐺', style: TextStyle(fontSize: 36)),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Привет, ${snapshot.displayName?.isNotEmpty == true ? snapshot.displayName : 'друг'}!',
                  style: const TextStyle(
                    fontSize: 21,
                    fontWeight: FontWeight.w900,
                    color: AppColors.text,
                  ),
                ),
                const SizedBox(height: 4),
                const Text(
                  'Рады видеть тебя в вашем уютном пространстве.',
                  style: TextStyle(height: 1.4, color: AppColors.text),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _StartCoupleCard extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: const Color(0xFFEFF7F1),
        borderRadius: BorderRadius.circular(22),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Пригласи своего человека',
            style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
          ),
          const SizedBox(height: 7),
          const Text(
            'Создай приглашение или присоединись по коду, чтобы открыть общий мир.',
            style: TextStyle(height: 1.45, color: AppColors.secondaryText),
          ),
          const SizedBox(height: 14),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: () => context.push('/couple-setup'),
              child: const Text('Настроить пару'),
            ),
          ),
        ],
      ),
    );
  }
}

class _CoupleCard extends StatelessWidget {
  final HomeSnapshot snapshot;

  const _CoupleCard({required this.snapshot});

  @override
  Widget build(BuildContext context) {
    final partners = snapshot.members
        .where((member) => member.email != snapshot.email)
        .toList();
    final partner = partners.isEmpty ? null : partners.first;

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppColors.white,
        borderRadius: BorderRadius.circular(22),
        border: Border.all(color: const Color(0xFFF0EAF4)),
      ),
      child: Row(
        children: [
          const CircleAvatar(
            backgroundColor: Color(0xFFF3EAF8),
            child: Text('💜'),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              partner == null
                  ? 'Ваша пара создана'
                  : 'Ваш мир вместе с ${partner.nickname?.isNotEmpty == true ? partner.nickname : partner.email}',
              style: const TextStyle(fontWeight: FontWeight.w800),
            ),
          ),
          const Icon(Icons.favorite_rounded, color: AppColors.lavenderDark),
        ],
      ),
    );
  }
}

class _PetStateCard extends StatelessWidget {
  final HomePet pet;

  const _PetStateCard({required this.pet});

  @override
  Widget build(BuildContext context) {
    final updatedAt = pet.updatedAt.toLocal();
    final updatedLabel =
        '${updatedAt.day.toString().padLeft(2, '0')}.'
        '${updatedAt.month.toString().padLeft(2, '0')}.'
        '${updatedAt.year}';
    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: const Color(0xFFFFF5E9),
        borderRadius: BorderRadius.circular(22),
      ),
      child: Row(
        children: [
          const Icon(Icons.pets_rounded, size: 32, color: Color(0xFFB77C4C)),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  'Питомец пары',
                  style: TextStyle(fontWeight: FontWeight.w900),
                ),
                const SizedBox(height: 4),
                Text(
                  pet.appearance.isEmpty
                      ? 'Внешность питомца ещё не настроена'
                      : 'Состояние питомца сохранено',
                  style: const TextStyle(
                    color: AppColors.secondaryText,
                    fontSize: 13,
                  ),
                ),
                const SizedBox(height: 3),
                Text(
                  'Обновлено $updatedLabel · версия ${pet.version}',
                  style: const TextStyle(
                    color: AppColors.secondaryText,
                    fontSize: 11,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _HomeUpcomingCard extends StatelessWidget {
  final List<HomeUpcomingItem> items;
  final bool isLoading;
  final Object? error;
  final VoidCallback onRetry;
  final VoidCallback onTap;

  const _HomeUpcomingCard({
    required this.items,
    required this.isLoading,
    required this.error,
    required this.onRetry,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) => Card(
    child: Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          InkWell(
            onTap: onTap,
            child: const Row(
              children: [
                Expanded(
                  child: Text(
                    'Ближайшие планы',
                    style: TextStyle(fontWeight: FontWeight.w900),
                  ),
                ),
                Icon(Icons.chevron_right_rounded),
              ],
            ),
          ),
          const SizedBox(height: 8),
          if (isLoading && items.isEmpty)
            const LinearProgressIndicator()
          else if (error != null && items.isEmpty)
            Row(
              children: [
                Expanded(child: Text(apiErrorMessage(error!))),
                IconButton(
                  onPressed: onRetry,
                  tooltip: 'Повторить',
                  icon: const Icon(Icons.refresh_rounded),
                ),
              ],
            )
          else if (items.isEmpty)
            const Text(
              'Нет задач со сроком, ближайших событий или дат.',
              style: TextStyle(color: AppColors.secondaryText),
            )
          else
            ...items.map(
              (item) => ListTile(
                contentPadding: EdgeInsets.zero,
                dense: true,
                leading: Icon(_iconForKind(item.kind)),
                title: Text(
                  item.title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                subtitle: Text(
                  '${_homeUpcomingDate(item.date)} · ${_labelForKind(item.kind)}',
                ),
              ),
            ),
        ],
      ),
    ),
  );

  static IconData _iconForKind(String kind) => switch (kind) {
    'task' => Icons.checklist_rounded,
    'event' => Icons.event_rounded,
    _ => Icons.cake_outlined,
  };

  static String _labelForKind(String kind) => switch (kind) {
    'task' => 'Задача',
    'event' => 'Событие',
    _ => 'Важная дата',
  };
}

class _SyncStatusCard extends StatelessWidget {
  final bool isSyncing;
  final Object? error;
  final VoidCallback onRetry;

  const _SyncStatusCard({
    required this.isSyncing,
    required this.error,
    required this.onRetry,
  });

  @override
  Widget build(BuildContext context) => Card(
    child: Padding(
      padding: const EdgeInsets.all(14),
      child: Row(
        children: [
          if (isSyncing)
            const SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          else
            const Icon(Icons.sync_problem_rounded),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              isSyncing
                  ? 'Синхронизация данных…'
                  : 'Не удалось синхронизировать: '
                        '${apiErrorMessage(error!)}',
              style: const TextStyle(fontSize: 13),
            ),
          ),
          if (!isSyncing)
            IconButton(
              onPressed: onRetry,
              tooltip: 'Повторить синхронизацию',
              icon: const Icon(Icons.refresh_rounded),
            ),
        ],
      ),
    ),
  );
}

class _LatestMomentCard extends StatelessWidget {
  final Map<String, dynamic> moment;
  final VoidCallback onTap;

  const _LatestMomentCard({required this.moment, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final title = moment['title'] as String? ?? 'Момент';
    final caption = moment['caption'] as String? ?? '';
    final happenedAt = DateTime.parse(
      moment['happened_at'] as String,
    ).toLocal();
    return Card(
      child: ListTile(
        onTap: onTap,
        leading: const Icon(Icons.auto_awesome_rounded),
        title: Text('Последний момент · $title'),
        subtitle: Text(
          caption.isEmpty
              ? _homeUpcomingDate(happenedAt)
              : '$caption · ${_homeUpcomingDate(happenedAt)}',
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
        ),
        trailing: const Icon(Icons.chevron_right_rounded),
      ),
    );
  }
}

class _PendingCoupleCard extends StatelessWidget {
  const _PendingCoupleCard();

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.all(20),
    decoration: BoxDecoration(
      color: const Color(0xFFFFF2E5),
      borderRadius: BorderRadius.circular(22),
    ),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          'Ждём второго участника',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
        ),
        const SizedBox(height: 7),
        const Text(
          'Создайте новое приглашение и отправьте партнёру код.',
          style: TextStyle(height: 1.45, color: AppColors.secondaryText),
        ),
        const SizedBox(height: 14),
        SizedBox(
          width: double.infinity,
          child: FilledButton(
            onPressed: () => context.push('/couple-create'),
            child: const Text('Открыть приглашение'),
          ),
        ),
      ],
    ),
  );
}

class _FeatureCard extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final Color color;
  final VoidCallback onTap;

  const _FeatureCard({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.color,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(20),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: AppColors.white,
          borderRadius: BorderRadius.circular(20),
          border: Border.all(color: const Color(0xFFF0EAF4)),
        ),
        child: Row(
          children: [
            Container(
              width: 46,
              height: 46,
              decoration: BoxDecoration(
                color: color.withValues(alpha: 0.45),
                borderRadius: BorderRadius.circular(15),
              ),
              child: Icon(icon, color: AppColors.text),
            ),
            const SizedBox(width: 13),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    title,
                    style: const TextStyle(fontWeight: FontWeight.w800),
                  ),
                  const SizedBox(height: 3),
                  Text(
                    subtitle,
                    style: const TextStyle(
                      fontSize: 12,
                      color: AppColors.secondaryText,
                    ),
                  ),
                ],
              ),
            ),
            const Icon(Icons.chevron_right_rounded),
          ],
        ),
      ),
    );
  }
}

class _SectionHeading extends StatelessWidget {
  final String title;

  const _SectionHeading({required this.title});

  @override
  Widget build(BuildContext context) => Text(
    title,
    style: const TextStyle(fontSize: 19, fontWeight: FontWeight.w900),
  );
}

class _ErrorCard extends StatelessWidget {
  final Object error;
  final VoidCallback onRetry;

  const _ErrorCard({required this.error, required this.onRetry});

  @override
  Widget build(BuildContext context) => Center(
    child: Padding(
      padding: const EdgeInsets.only(top: 80),
      child: Column(
        children: [
          const Icon(Icons.cloud_off_rounded, size: 44),
          const SizedBox(height: 12),
          Text(apiErrorMessage(error), textAlign: TextAlign.center),
          const SizedBox(height: 12),
          FilledButton(onPressed: onRetry, child: const Text('Повторить')),
        ],
      ),
    ),
  );
}

String _homeUpcomingDate(DateTime date) =>
    '${date.day.toString().padLeft(2, '0')}.'
    '${date.month.toString().padLeft(2, '0')}';
