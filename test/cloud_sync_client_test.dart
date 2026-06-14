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
}
