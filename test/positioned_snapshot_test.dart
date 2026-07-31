import 'package:flutter_test/flutter_test.dart';
import 'package:simple_present/sync/positioned_snapshot.dart';

void main() {
  PositionedSnapshotItem<String> item(String value, int position) =>
      PositionedSnapshotItem(value: value, id: value, position: position);

  test('reconstructs server order from an empty local list', () {
    final merged = mergePositionedSnapshot(
      existing: <String>[],
      incoming: <PositionedSnapshotItem<String>>[
        item('third', 2),
        item('first', 0),
        item('second', 1),
      ],
      idOf: (value) => value,
    );

    expect(merged, <String>['first', 'second', 'third']);
  });

  test('replaces a differently ordered local snapshot', () {
    final merged = mergePositionedSnapshot(
      existing: <String>['third', 'first', 'second'],
      incoming: <PositionedSnapshotItem<String>>[
        item('first', 0),
        item('second', 1),
        item('third', 2),
      ],
      idOf: (value) => value,
    );

    expect(merged, <String>['first', 'second', 'third']);
  });

  test('places a partial update without dropping unchanged items', () {
    final merged = mergePositionedSnapshot(
      existing: <String>['first', 'second', 'third'],
      incoming: <PositionedSnapshotItem<String>>[
        item('third', 0),
      ],
      idOf: (value) => value,
    );

    expect(merged, <String>['third', 'first', 'second']);
  });
}
