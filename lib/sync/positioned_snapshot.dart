class PositionedSnapshotItem<T> {
  const PositionedSnapshotItem({
    required this.value,
    required this.id,
    required this.position,
  });

  final T value;
  final String id;
  final int position;
}

List<T> mergePositionedSnapshot<T>({
  required List<T> existing,
  required List<PositionedSnapshotItem<T>> incoming,
  required String Function(T value) idOf,
}) {
  final incomingIds = incoming.map((item) => item.id).toSet();
  final merged = existing
      .where((item) => !incomingIds.contains(idOf(item)))
      .toList(growable: true);
  final sorted = List<PositionedSnapshotItem<T>>.from(incoming)
    ..sort((a, b) {
      final byPosition = a.position.compareTo(b.position);
      return byPosition != 0 ? byPosition : a.id.compareTo(b.id);
    });

  for (final item in sorted) {
    merged.insert(item.position.clamp(0, merged.length), item.value);
  }
  return merged;
}
