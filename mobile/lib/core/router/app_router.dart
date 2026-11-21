import 'package:go_router/go_router.dart';

import '../../features/auth/presentation/screens/login_screen.dart';
import '../../features/auth/presentation/screens/register_screen.dart';
import '../../features/home/splash_screen.dart';
import '../../features/auth/presentation/screens/forgot_password_screen.dart';
import '../../features/auth/presentation/screens/profile_screen.dart';
import '../../features/onboarding/character_screen.dart';
import '../../features/onboarding/presentation/onboarding_screen.dart';
import '../../features/onboarding/profile_setup_screen.dart';
import '../../features/onboarding/couple_intro_screen.dart';
import '../../features/couple/couple_setup_screen.dart';
import '../../features/couple/couple_create_screen.dart';
import '../../features/couple/couple_join_screen.dart';
import '../../features/couple/couple_scan_screen.dart';
import '../../features/home/main_shell.dart';
import '../../features/pet/presentation/pet_screen.dart';
import '../../features/notifications/presentation/notifications_screen.dart';
import '../../features/games/presentation/games_screen.dart';
import '../../features/wheel/presentation/wheel_screen.dart';
import '../../features/chat/presentation/chat_screen.dart';
import '../../features/letters/presentation/letters_screen.dart';
import '../../features/location/presentation/location_screen.dart';
import '../../features/dates/presentation/dates_screen.dart';

final appRouter = GoRouter(
  initialLocation: '/',
  routes: [
    GoRoute(path: '/', builder: (context, state) => const SplashScreen()),
    GoRoute(path: '/app', builder: (context, state) => const MainShell()),
    GoRoute(
      path: '/profile',
      builder: (context, state) => const ProfileScreen(),
    ),
    GoRoute(
      path: '/onboarding-tests',
      builder: (context, state) => const OnboardingScreen(),
    ),
    GoRoute(path: '/login', builder: (context, state) => const LoginScreen()),
    GoRoute(
      path: '/register',
      builder: (context, state) => const RegisterScreen(),
    ),
    GoRoute(
      path: '/forgot-password',
      builder: (context, state) => const ForgotPasswordScreen(),
    ),
    GoRoute(
      path: '/character',
      builder: (context, state) => const CharacterScreen(),
    ),
    GoRoute(
      path: '/profile-setup',
      builder: (context, state) {
        final data = state.extra as Map<String, dynamic>;

        return ProfileSetupScreen(
          character: data['character'] as String,
          characterName: data['characterName'] as String,
        );
      },
    ),
    GoRoute(
      path: '/couple-intro',
      builder: (context, state) {
        final data = state.extra as Map<String, dynamic>;

        return CoupleIntroScreen(
          name: data['name'] as String,
          about: data['about'] as String,
          character: data['character'] as String,
          characterName: data['characterName'] as String,
          color: data['color'] as int,
        );
      },
    ),
    GoRoute(
      path: '/couple-setup',
      builder: (context, state) => const CoupleSetupScreen(),
    ),
    GoRoute(
      path: '/couple-create',
      builder: (context, state) => const CoupleCreateScreen(),
    ),
    GoRoute(
      path: '/couple-join',
      builder: (context, state) => const CoupleJoinScreen(),
    ),
    GoRoute(
      path: '/couple-scan',
      builder: (context, state) => const CoupleScanScreen(),
    ),
    GoRoute(
      path: '/pet',
      builder: (context, state) => const PetScreen(),
    ),
    GoRoute(
      path: '/notifications',
      builder: (context, state) => const NotificationsScreen(),
    ),
    GoRoute(
      path: '/games',
      builder: (context, state) => const GamesScreen(),
    ),
    GoRoute(
      path: '/wheel',
      builder: (context, state) => const WheelScreen(),
    ),
    GoRoute(
      path: '/chat',
      builder: (context, state) => const ChatScreen(),
    ),
    GoRoute(
      path: '/letters',
      builder: (context, state) => const LettersScreen(),
    ),
    GoRoute(
      path: '/location',
      builder: (context, state) => const LocationScreen(),
    ),
    GoRoute(
      path: '/dates',
      builder: (context, state) => const DatesScreen(),
    ),
  ],
);
