import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'network/api_client.dart';
import 'storage/token_storage.dart';
import 'storage/onboarding_preferences.dart';
import 'storage/local_database.dart';
import 'sync/sync_api.dart';
import 'sync/sync_service.dart';
import '../features/auth/data/auth_api.dart';
import '../features/auth/data/profile_api.dart';
import '../features/couple/data/couple_api.dart';
import '../features/home/data/home_api.dart';
import '../features/habits/data/habits_api.dart';
import '../features/planning/data/planning_api.dart';
import '../features/wellness/data/wellness_api.dart';
import '../features/moments/data/moments_api.dart';
import '../features/moments/data/media_api.dart';
import '../features/onboarding/data/onboarding_api.dart';
import '../features/pet/data/pet_api.dart';
import '../features/notifications/data/notifications_api.dart';
import '../features/games/data/games_api.dart';
import '../features/wheel/data/wheel_api.dart';
import '../features/chat/data/chat_api.dart';
import '../features/letters/data/letters_api.dart';
import '../features/location/data/location_api.dart';
import '../features/dates/data/dates_api.dart';

final tokenStorageProvider = Provider<TokenStorage>((ref) => TokenStorage());

final onboardingPreferencesProvider = Provider<OnboardingPreferences>(
  (ref) => OnboardingPreferences(),
);

final currentUserIdProvider = StateProvider<String?>((ref) => null);

final apiClientProvider = Provider<ApiClient>(
  (ref) => ApiClient(ref.watch(tokenStorageProvider)),
);

final localDatabaseProvider = Provider<LocalDatabase>((ref) {
  final database = LocalDatabase();
  ref.onDispose(database.close);
  return database;
});

final syncApiProvider = Provider<SyncApi>(
  (ref) => SyncApi(ref.watch(apiClientProvider)),
);

final syncServiceProvider = Provider<SyncService>(
  (ref) =>
      SyncService(ref.watch(syncApiProvider), ref.watch(localDatabaseProvider)),
);

final authApiProvider = Provider<AuthApi>(
  (ref) => AuthApi(
    apiClient: ref.watch(apiClientProvider),
    tokenStorage: ref.watch(tokenStorageProvider),
  ),
);

final profileApiProvider = Provider<ProfileApi>(
  (ref) => ProfileApi(ref.watch(apiClientProvider)),
);

final coupleApiProvider = Provider<CoupleApi>(
  (ref) => CoupleApi(ref.watch(apiClientProvider)),
);

final homeApiProvider = Provider<HomeApi>(
  (ref) => HomeApi(ref.watch(apiClientProvider)),
);

final planningApiProvider = Provider<PlanningApi>(
  (ref) => PlanningApi(
    ref.watch(apiClientProvider),
    ref.watch(localDatabaseProvider),
  ),
);

final habitsApiProvider = Provider<HabitsApi>(
  (ref) => HabitsApi(ref.watch(apiClientProvider)),
);

final wellnessApiProvider = Provider<WellnessApi>(
  (ref) => WellnessApi(ref.watch(apiClientProvider)),
);

final momentsApiProvider = Provider<MomentsApi>(
  (ref) => MomentsApi(ref.watch(apiClientProvider)),
);

final mediaApiProvider = Provider<MediaApi>(
  (ref) => MediaApi(ref.watch(apiClientProvider)),
);

final onboardingApiProvider = Provider<OnboardingApi>(
  (ref) => OnboardingApi(ref.watch(apiClientProvider)),
);

final petApiProvider = Provider<PetApi>(
  (ref) => PetApi(ref.watch(apiClientProvider)),
);

final notificationsApiProvider = Provider<NotificationsApi>(
  (ref) => NotificationsApi(ref.watch(apiClientProvider)),
);

final gamesApiProvider = Provider<GamesApi>(
  (ref) => GamesApi(ref.watch(apiClientProvider)),
);

final wheelApiProvider = Provider<WheelApi>(
  (ref) => WheelApi(ref.watch(apiClientProvider)),
);

final chatApiProvider = Provider<ChatApi>(
  (ref) => ChatApi(ref.watch(apiClientProvider)),
);

final lettersApiProvider = Provider<LettersApi>(
  (ref) => LettersApi(ref.watch(apiClientProvider)),
);

final locationApiProvider = Provider<LocationApi>(
  (ref) => LocationApi(ref.watch(apiClientProvider)),
);

final datesApiProvider = Provider<DatesApi>(
  (ref) => DatesApi(ref.watch(apiClientProvider)),
);
