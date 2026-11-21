import 'package:flutter_test/flutter_test.dart';
import 'package:lovetea/features/couple/data/couple_api.dart';

void main() {
  group('invitationValueFromQr', () {
    test('extracts an invitation token from a link', () {
      expect(
        invitationValueFromQr('https://lovetea.example/join?token=abc123'),
        'abc123',
      );
    });

    test('extracts an invitation code from a link', () {
      expect(
        invitationValueFromQr('https://lovetea.example/join?code=123456'),
        '123456',
      );
    });

    test('trims a raw invitation value', () {
      expect(invitationValueFromQr('  abc123  '), 'abc123');
    });
  });

  test('parses a pending two-person detach confirmation', () {
    final result = CoupleDetachResult.fromJson({
      'couple_id': 'couple-1',
      'status': 'active',
      'confirmed': false,
      'message': 'Waiting for your partner.',
    });

    expect(result.status, 'active');
    expect(result.confirmed, isFalse);
  });

  test('parses completed detach confirmation', () {
    final result = CoupleDetachResult.fromJson({
      'couple_id': 'couple-1',
      'status': 'detached',
      'confirmed': true,
      'message': 'Both partners confirmed.',
    });

    expect(result.status, 'detached');
    expect(result.confirmed, isTrue);
  });
}
