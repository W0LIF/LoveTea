import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:lovetea/core/network/api_client.dart';
import 'package:lovetea/core/storage/token_storage.dart';
import 'package:lovetea/features/moments/data/media_api.dart';

void main() {
  test('parses upload metadata returned by the backend', () {
    final upload = MediaUploadResponse.fromJson({
      'media': {
        'id': 'media-1',
        'scope': 'personal',
        'content_type': 'image/png',
        'size_bytes': 120,
        'status': 'pending',
        'version': 1,
        'created_at': '2026-11-20T12:00:00Z',
      },
      'upload_url': 'http://localhost:9000/lovetea-media',
      'form_fields': {'key': 'media/media-1', 'policy': 'signed-policy'},
      'expires_at': '2026-11-20T12:10:00Z',
    });

    expect(upload.media.id, 'media-1');
    expect(upload.media.status, 'pending');
    expect(upload.formFields['key'], 'media/media-1');
  });

  test(
    'rejects unsupported image types and files over the backend size limit',
    () async {
      final api = MediaApi(ApiClient(TokenStorage()));

      await expectLater(
        api.uploadImage(
          bytes: Uint8List.fromList([1]),
          fileName: 'image.webp',
          contentType: 'image/webp',
          scope: 'personal',
        ),
        throwsArgumentError,
      );
      await expectLater(
        api.uploadImage(
          bytes: Uint8List(maxMediaUploadBytes + 1),
          fileName: 'large.jpg',
          contentType: 'image/jpeg',
          scope: 'personal',
        ),
        throwsArgumentError,
      );
    },
  );
}
