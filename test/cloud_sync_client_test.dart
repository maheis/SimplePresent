import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:simple_present/sync/cloud_sync_client.dart';

void main() {
  test('normalizeWordPhrase enforces 9 words', () {
    final normalized = CloudSyncClient.normalizeWordPhrase(
      '  Apfel   berg fluss stern Wald nebel licht fenster uhr  ',
    );

    expect(
      normalized,
      'apfel berg fluss stern wald nebel licht fenster uhr',
    );
  });

  test('derivePairingKeyPair is deterministic for same phrase', () async {
    const phrase = 'apfel berg fluss stern wald nebel licht fenster uhr';
    final k1 = await CloudSyncClient.derivePairingKeyPair(phrase);
    final k2 = await CloudSyncClient.derivePairingKeyPair(phrase);

    final p1 = await k1.extractPublicKey();
    final p2 = await k2.extractPublicKey();

    expect(p1.bytes, p2.bytes);
  });

  test('pushItems exposes HTTP 409 as sync conflict', () async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    addTearDown(() => server.close(force: true));
    server.listen((request) async {
      await utf8.decoder.bind(request).join();
      request.response
        ..statusCode = HttpStatus.conflict
        ..write('item conflict');
      await request.response.close();
    });

    final client = CloudSyncClient(
      serverBaseUrl: 'http://${server.address.address}:${server.port}',
    );

    await expectLater(
      client.pushItems(
        accountId: 'account-a',
        token: 'token-a',
        items: <Map<String, dynamic>>[
          <String, dynamic>{
            'id': 'task:1',
            'payload': <String, dynamic>{'value': 'stale'},
            'modified_at': 1,
            'version': 1,
          },
        ],
      ),
      throwsA(
        isA<CloudSyncException>()
            .having((error) => error.statusCode, 'statusCode', 409)
            .having((error) => error.path, 'path', '/push')
            .having((error) => error.isConflict, 'isConflict', isTrue),
      ),
    );
  });
}
