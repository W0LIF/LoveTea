import 'dart:typed_data';

import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';

const maxMediaUploadBytes = 20 * 1024 * 1024;

class MediaFile {
  final String id;
  final String scope;
  final String contentType;
  final int sizeBytes;
  final String status;
  final int version;

  const MediaFile({
    required this.id,
    required this.scope,
    required this.contentType,
    required this.sizeBytes,
    required this.status,
    required this.version,
  });

  factory MediaFile.fromJson(Map<String, dynamic> json) => MediaFile(
    id: json['id'] as String,
    scope: json['scope'] as String,
    contentType: json['content_type'] as String,
    sizeBytes: (json['size_bytes'] as num).toInt(),
    status: json['status'] as String,
    version: (json['version'] as num).toInt(),
  );
}

class MediaUploadResponse {
  final MediaFile media;
  final String uploadUrl;
  final Map<String, String> formFields;

  const MediaUploadResponse({
    required this.media,
    required this.uploadUrl,
    required this.formFields,
  });

  factory MediaUploadResponse.fromJson(Map<String, dynamic> json) =>
      MediaUploadResponse(
        media: MediaFile.fromJson(
          Map<String, dynamic>.from(json['media'] as Map),
        ),
        uploadUrl: json['upload_url'] as String,
        formFields: Map<String, String>.from(json['form_fields'] as Map),
      );
}

class MediaApi {
  final ApiClient _apiClient;

  MediaApi(this._apiClient);

  Dio get _dio => _apiClient.dio;

  Future<MediaFile> uploadImage({
    required Uint8List bytes,
    required String fileName,
    required String contentType,
    required String scope,
    void Function(int sent, int total)? onSendProgress,
  }) async {
    if (bytes.isEmpty || bytes.length > maxMediaUploadBytes) {
      throw ArgumentError.value(
        bytes.length,
        'bytes.length',
        'Image size must be between 1 byte and 20 MiB.',
      );
    }
    if (contentType != 'image/jpeg' && contentType != 'image/png') {
      throw ArgumentError.value(
        contentType,
        'contentType',
        'Only JPEG and PNG images are supported.',
      );
    }

    final response = await _dio.post<Map<String, dynamic>>(
      '/media/uploads',
      data: {'scope': scope, 'content_type': contentType},
    );
    final upload = MediaUploadResponse.fromJson(response.data!);
    final uri = Uri.parse(upload.uploadUrl);
    if (uri.scheme != 'http' && uri.scheme != 'https') {
      throw const FormatException('Ожидался HTTP URL для загрузки фото.');
    }

    final form = FormData.fromMap({
      ...upload.formFields,
      'file': MultipartFile.fromBytes(
        bytes,
        filename: fileName,
        contentType: DioMediaType.parse(contentType),
      ),
    });

    // The signed form is sufficient authorization; never send API bearer tokens
    // to the object-storage host.
    await Dio(
      BaseOptions(
        connectTimeout: const Duration(seconds: 15),
        sendTimeout: const Duration(seconds: 90),
        receiveTimeout: const Duration(seconds: 15),
      ),
    ).post<void>(
      upload.uploadUrl,
      data: form,
      options: Options(contentType: Headers.multipartFormDataContentType),
      onSendProgress: onSendProgress,
    );

    final completed = await _dio.post<Map<String, dynamic>>(
      '/media/${upload.media.id}/complete',
    );
    return MediaFile.fromJson(completed.data!);
  }

  Future<String> createDownloadUrl(String mediaId) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/media/$mediaId/download',
    );
    final data = response.data;
    if (data == null || data['url'] is! String) {
      throw const FormatException('Ожидался URL для просмотра фото.');
    }
    return data['url'] as String;
  }

  Future<void> deleteMedia(MediaFile media) async {
    await _dio.delete<void>(
      '/media/${Uri.encodeComponent(media.id)}',
      queryParameters: {'version': media.version},
    );
  }
}
