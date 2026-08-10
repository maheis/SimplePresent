# UI-Design-Richtlinie: SimplePresent

Diese Richtlinie beschreibt das visuelle Design von SimplePresent als Referenz für neue Apps und als Anweisung zur konsistenten Umsetzung.

---

## Leitbild

Minimalistisch, dunkel, zugänglich. Die App tritt visuell zurück — Inhalte (Aufgaben, Text) stehen im Vordergrund. Farbe wird sparsam eingesetzt und trägt ausschließlich semantische Bedeutung (Status, Dringlichkeit).

---

## Theme

| Eigenschaft | Wert |
|---|---|
| Helligkeit | Dark only (`Brightness.dark`) |
| Seed-Farbe | `Colors.teal` |
| Erzeugung | `ColorScheme.fromSeed(seedColor: Colors.teal, brightness: Brightness.dark)` |
| Flutter-Widget | `ThemeData` mit `colorScheme`, kein `primarySwatch` |

Das Theme wird komplett über `ColorScheme.fromSeed` generiert — keine einzelnen Farb-Overrides. Die App hat keinen Light Mode.

---

## Farben

### Strukturfarben (aus ColorScheme)

Alle Hintergründe, Oberflächen und Standard-Text kommen aus dem generierten `ColorScheme` — nie hartcodierte Hex-Werte für strukturelle UI.

| Rolle | Flutter-Token | Verwendung |
|---|---|---|
| Hintergrund | `cs.surface` | Seiten-Hintergrund |
| Karten/Container | `cs.primaryContainer`, `cs.secondaryContainer` | Task-Karten, Panels |
| Primärtext | `cs.onSurface` | Standard-Textfarbe |
| Sekundärtext | `cs.onSurfaceVariant` | Datum, Metadaten, Hinweise |

### Semantische Farben (hartcodiert)

Diese Farben haben eine feste Bedeutung und werden konsistent über die ganze App verwendet:

| Bedeutung | Farbe | Flutter-Wert |
|---|---|---|
| Aufgabe: wichtig | Amber | `Colors.amber` |
| Aufgabe: in Bearbeitung | Hellgrün | `Color(0xFF9EEB9E)` |
| Aufgabe: erledigt | Gedimmt | `cs.onSurface.withAlpha((0.6 * 255).round())` |
| Status: OK / Sync erfolgreich | Hellgrün | `Colors.lightGreenAccent` / `Colors.greenAccent` |
| Status: Warnung / Beschäftigt | Amber/Orange | `Colors.amberAccent` / `Colors.orangeAccent` |
| Status: Fehler | Rot | `Colors.redAccent` |
| Termin: rechtzeitig | Grün | `Colors.green` |
| Termin: überfällig | Rot | `Colors.red` |
| Transparent / kein Hintergrund | — | `Colors.transparent` |

**Regel:** Statusfarben niemals als Hintergrundfarbe — immer als Textfarbe oder Icon-Farbe einsetzen.

---

## Typografie

### Schriftarten

Die App bietet drei Fonts zur Auswahl — OpenDyslexic ist Standard (Barrierefreiheit first):

| Font | Einsatz |
|---|---|
| **OpenDyslexic** | Standard — verbesserte Lesbarkeit für Dyslexiker |
| NotoSans | Saubere Alternative, System-nah |
| CourierPrime | Monospace-Option |

Alle Fonts werden über `pubspec.yaml` als Asset eingebunden (`assets/fonts/`). Der gewählte Font wird via `_fontFamily` State-Variable global durchgereicht und auf das `ThemeData.textTheme` angewendet.

```dart
// Font auf das gesamte Theme anwenden
data: baseTheme.copyWith(
  textTheme: baseTheme.textTheme.apply(fontFamily: _fontFamily),
  primaryTextTheme: baseTheme.primaryTextTheme.apply(fontFamily: _fontFamily),
)
```

### Größenskala

| Verwendung | Größe | Gewicht |
|---|---|---|
| Screen-Titel / Hero | 24 | `w700` |
| Abschnitts-Überschrift | 18–19 | `bold` / `w700` |
| Standard-Body / Task-Titel | 15–16 | normal / `w600` |
| Sekundär / Label | 12–14 | normal |
| Kleintext / Metadaten | 10–11 | normal |

Die Basis-Schriftgröße (`_baseFontSize`) liegt bei **15.0** und wird für Task-Text verwendet.

---

## Abstände (Spacing)

Kein eigenes Spacing-System — es gelten folgende bevorzugte Werte in der Praxis:

| Token | Wert | Typische Verwendung |
|---|---|---|
| xs | 4 px | Minimaler Innenabstand, Icon-Gaps |
| sm | 6–8 px | Standard-Padding horizontal |
| md | 12 px | Standard-Padding vertikal, Listenabstände |
| lg | 16 px | Abschnitte, größere Container |

Häufigste Padding-Werte: `horizontal: 12`, `vertical: 6–8`.

---

## Eckenradien

| Element | Radius |
|---|---|
| Buttons, Chips, kleine Karten | `BorderRadius.circular(12)` |
| Größere Container / Dialoge | `BorderRadius.circular(14)` |

---

## Icons

Ausschließlich **Material Icons** (`Icons.*`) aus dem Flutter-Standard-Set. Kein eigenes Icon-Set, keine externen Icon-Pakete.

Bevorzugt werden `_rounded`-Varianten wenn vorhanden (z. B. `Icons.download_rounded`, `Icons.upload_rounded`).

Wichtige Icons und ihre Bedeutung in der App:

| Icon | Bedeutung |
|---|---|
| `Icons.radio_button_unchecked` | Aufgabe offen |
| `Icons.task_alt` | Aufgabe erledigt |
| `Icons.star` / `Icons.star_border` | Wichtig (Flag) |
| `Icons.play_arrow` / `Icons.stop` | In Bearbeitung starten/stoppen |
| `Icons.push_pin` / `Icons.push_pin_outlined` | Angeheftet |
| `Icons.drag_handle` | Drag & Drop Handle |
| `Icons.calendar_today` | Datum |
| `Icons.timelapse` | Zeiterfassung |
| `Icons.delete` / `Icons.delete_forever` | Löschen / endgültig löschen |
| `Icons.sync` | Cloud-Sync |
| `Icons.more_vert` | Kontextmenü |

---

## Komponenten-Konventionen

### Task-Zeile

- Hintergrund: kein farbiger Hintergrund — nur der Kartencontainer (`Card` oder `ColorScheme`-Container)
- Textfarbe zeigt den Status (siehe semantische Farben)
- Linke Seite: Status-Icon (offen/erledigt)
- Rechte Seite: Aktions-Icons (`IconButton` mit `tapTargetSize: MaterialTapTargetSize.shrinkWrap`)
- Gedimmter Text bei erledigten Aufgaben (60 % Opacity)

### Buttons

- Primäraktion: `ElevatedButton`
- Sekundäraktion / Abbrechen: `TextButton`
- Ikonbasierte Aktionen in Listen: `IconButton`
- Destruktive Aktionen (Löschen): `foregroundColor: Colors.redAccent`, `BorderSide(color: Colors.redAccent)`

### Dialoge

- Standard Flutter `AlertDialog` / `showDialog`
- Keine Custom-Dialog-Rahmen

### Listen

- `ListView` ohne Trennlinien (kein `Divider`)
- Drag & Drop via `ReorderableListView` mit `Icons.drag_handle`

---

## Layout

- **3-Panel-Struktur:** Today | Backlog | Done (horizontal scrollbar / Navigation)
- Kein AppBar im klassischen Sinne — Titel steht als Text im Panel
- Kein Floating Action Button auf Hauptebene
- Statuszeile (Sync-Status, Uhrzeit) oben im Panel, klein (fontSize 11)

---

## Barrierefreiheit

- **OpenDyslexic als Standard-Font** — aktiv als erstes Accessibility-Feature
- Schriftgröße benutzerkonfigurierbar (Settings)
- Kontraste folgen dem generierten Dark ColorScheme (Material Design-konform)
- Alle `IconButton`s haben `tooltip`
- Keine reinen Farbsignale — Icons begleiten immer Farbänderungen
