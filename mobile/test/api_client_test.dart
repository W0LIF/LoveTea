import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:lovetea/core/network/api_client.dart';
import 'package:lovetea/core/storage/token_storage.dart';

void main() {
  test('refreshes an expired access token and retries the request', () async {
    final storage = _MemoryTokenStorage();
    final adapter = _RefreshingAdapter();
    final client = ApiClient(storage);
    client.dio.httpClientAdapter = adapter;

    final response = await client.dio.get<Map<String, dynamic>>('/private');

    expect(response.data, {'ok': true});
    expect(adapter.refreshCount, 1);
    expect(adapter.privateRequestCount, 2);
    expect(storage.accessToken, 'fresh-access');
    expect(storage.refreshToken, 'rotated-refresh');
  });
}

class _MemoryTokenStorage extends TokenStorage {
  String? accessToken = 'expired-access';
  String? refreshToken = 'current-refresh';

  @override
  Future<String?> getAccessToken() async => accessToken;

  @override
  Future<String?> getRefreshToken() async => refreshToken;

  @override
  Future<void> saveTokens({
    required String accessToken,
    required String refreshToken,
  }) async {
    this.accessToken = accessToken;
    this.refreshToken = refreshToken;
  }

  @override
  Future<void> clearTokens() async {
    accessToken = null;
    refreshToken = null;
  }
}

class _RefreshingAdapter implements HttpClientAdapter {
  int refreshCount = 0;
  int privateRequestCount = 0;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    if (options.uri.path.endsWith('/auth/refresh')) {
      refreshCount++;
      return ResponseBody.fromString(
        jsonEncode({
          'access_token': 'fresh-access',
          'refresh_token': 'rotated-refresh',
        }),
        200,
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
        },
      );
    }

    privateRequestCount++;
    if (options.headers['Authorization'] != 'Bearer fresh-access') {
      return ResponseBody.fromString(
        jsonEncode({'error': 'unauthorized'}),
        401,
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
        },
      );
    }

    return ResponseBody.fromString(
      jsonEncode({'ok': true}),
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}
