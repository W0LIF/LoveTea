import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';

class CoupleInvitation {
  final String coupleId;
  final String code;
  final String token;
  final DateTime expiresAt;

  const CoupleInvitation({
    required this.coupleId,
    required this.code,
    required this.token,
    required this.expiresAt,
  });
}

class CoupleDetachResult {
  final String status;
  final bool confirmed;

  const CoupleDetachResult({required this.status, required this.confirmed});

  factory CoupleDetachResult.fromJson(Map<String, dynamic> json) =>
      CoupleDetachResult(
        status: json['status'] as String,
        confirmed: json['confirmed'] as bool,
      );
}

class CoupleApi {
  final ApiClient _apiClient;

  CoupleApi(this._apiClient);

  Dio get _dio => _apiClient.dio;

  Future<CoupleInvitation> createInvitation() async {
    late final String coupleId;
    try {
      final coupleResponse = await _dio.post('/couples');
      final couple = Map<String, dynamic>.from(coupleResponse.data as Map);
      coupleId = couple['couple_id'] as String;
    } on DioException catch (createError, stackTrace) {
      if (createError.response?.statusCode != 409) rethrow;

      try {
        final response = await _dio.get('/couples/me');
        final existing = Map<String, dynamic>.from(response.data as Map);
        if (existing['status'] != 'pending') {
          Error.throwWithStackTrace(createError, stackTrace);
        }
        coupleId = existing['couple_id'] as String;
      } on DioException {
        Error.throwWithStackTrace(createError, stackTrace);
      }
    }

    return _issueInvitation(coupleId);
  }

  Future<CoupleInvitation> renewInvitation(String coupleId) =>
      _issueInvitation(coupleId);

  Future<CoupleInvitation> _issueInvitation(String coupleId) async {
    final invitationResponse = await _dio.post(
      '/couples/$coupleId/invitations',
    );
    final invitation = Map<String, dynamic>.from(
      invitationResponse.data as Map,
    );

    return CoupleInvitation(
      coupleId: coupleId,
      code: invitation['code'] as String,
      token: invitation['token'] as String,
      expiresAt: DateTime.parse(invitation['expires_at'] as String),
    );
  }

  Future<void> joinByCode(String code) async {
    await _dio.post('/couples/join', data: {'code': code});
  }

  Future<void> joinByToken(String token) async {
    await _dio.post('/couples/join', data: {'token': token});
  }

  Future<void> joinByInvitation(String value) async {
    final invitation = invitationValueFromQr(value);
    if (RegExp(r'^\d{6}$').hasMatch(invitation)) {
      await joinByCode(invitation);
    } else {
      await joinByToken(invitation);
    }
  }

  Future<bool> hasPartner() async {
    final response = await _dio.get('/couples/me');
    final couple = Map<String, dynamic>.from(response.data as Map);
    final members = couple['members'];
    return couple['status'] == 'active' &&
        members is List &&
        members.length == 2;
  }

  Future<void> updateMemberProfile({
    required String nickname,
    required String mascotType,
  }) async {
    await _dio.patch(
      '/couples/me/profile',
      data: {'nickname': nickname, 'mascot_type': mascotType},
    );
  }

  Future<CoupleDetachResult> confirmDetach() async {
    final response = await _dio.post('/couples/me/detach/confirm');
    return CoupleDetachResult.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }
}

String invitationValueFromQr(String rawValue) {
  final value = rawValue.trim();
  final uri = Uri.tryParse(value);
  final token = uri?.queryParameters['token']?.trim();
  if (token != null && token.isNotEmpty) return token;

  final code = uri?.queryParameters['code']?.trim();
  if (code != null && code.isNotEmpty) return code;

  return value;
}
