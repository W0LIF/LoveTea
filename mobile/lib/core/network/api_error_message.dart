import 'package:dio/dio.dart';

String apiErrorMessage(Object error) {
  if (error is DioException) {
    final responseData = error.response?.data;
    if (responseData is Map && responseData['error'] is String) {
      return _translateApiError(responseData['error'] as String);
    }

    if (error.response == null) {
      return 'Не удалось подключиться к серверу. Проверьте адрес API и сеть.';
    }

    return 'Сервер вернул ошибку (${error.response?.statusCode ?? 'неизвестно'}).';
  }

  return 'Не удалось выполнить запрос. Попробуйте ещё раз.';
}

String _translateApiError(String message) {
  switch (message) {
    case 'invalid email or password':
      return 'Неверная почта или пароль.';
    case 'email already registered':
      return 'Аккаунт с такой почтой уже существует.';
    case 'user already belongs to a couple':
      return 'Ваш аккаунт уже состоит в паре.';
    case 'invitation not found':
      return 'Приглашение не найдено. Проверьте код.';
    case 'invitation has expired':
      return 'Срок действия приглашения истёк.';
    case 'cannot accept your own invitation':
      return 'Нельзя присоединиться к собственному приглашению.';
    case 'too many attempts':
    case 'too many attempts; try again later':
      return 'Слишком много попыток. Попробуйте позже.';
    case 'an active couple is required':
      return 'Сначала подключите партнёра.';
    case 'invalid content data':
      return 'Проверьте данные момента и прикреплённые фото.';
    case 'content not found':
      return 'Момент или фото больше недоступны.';
    case 'media upload is not complete':
      return 'Фото ещё загружается. Попробуйте сохранить момент позже.';
    case 'resource version conflict':
      return 'Данные изменились. Обновите экран и попробуйте снова.';
    default:
      return 'Не удалось выполнить запрос: $message';
  }
}
