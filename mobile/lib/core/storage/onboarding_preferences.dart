import 'package:shared_preferences/shared_preferences.dart';

class OnboardingPreferences {
  static const _mascotTypeKey = 'onboarding_mascot_type';
  static const _mascotNicknameKey = 'onboarding_mascot_nickname';

  Future<void> saveMascot({
    required String type,
    required String nickname,
  }) async {
    final preferences = await SharedPreferences.getInstance();
    await preferences.setString(_mascotTypeKey, type);
    await preferences.setString(_mascotNicknameKey, nickname);
  }

  Future<({String type, String nickname})?> getMascot() async {
    final preferences = await SharedPreferences.getInstance();
    final type = preferences.getString(_mascotTypeKey);
    final nickname = preferences.getString(_mascotNicknameKey);
    if (type == null || nickname == null) return null;
    return (type: type, nickname: nickname);
  }

  Future<void> clearMascot() async {
    final preferences = await SharedPreferences.getInstance();
    await preferences.remove(_mascotTypeKey);
    await preferences.remove(_mascotNicknameKey);
  }
}
