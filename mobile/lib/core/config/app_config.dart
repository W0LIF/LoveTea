import 'package:flutter/foundation.dart';

class AppConfig {
  static String get baseUrl {
    const configuredUrl = String.fromEnvironment('API_BASE_URL');
    if (configuredUrl.isNotEmpty) {
      return configuredUrl.replaceFirst(RegExp(r'/$'), '');
    }

    if (kIsWeb) {
      return 'http://localhost:8080';
    }

    if (defaultTargetPlatform == TargetPlatform.android) {
      // Android Emulator обращается к компьютеру через 10.0.2.2
      return 'http://10.0.2.2:8080';
    }

    return 'http://localhost:8080';
  }

  static const String apiPrefix = '/api/v1';
}
