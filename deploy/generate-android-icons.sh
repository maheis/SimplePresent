#!/usr/bin/env bash
set -euo pipefail

# Generates Android launcher and notification icons from SVGs in assets/icons.
# Generates density PNGs plus the adaptive XML background drawable.
# Usage: ./generate-android-icons.sh

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SVG_DIR="$PROJECT_ROOT/assets/icons"
ANDROID_RES="$PROJECT_ROOT/android/app/src/main/res"

SVG_BACKGROUND="$SVG_DIR/color_teal_icon_android_background.svg"
SVG_FOREGROUND="$SVG_DIR/color_transparent_icon_android.svg"
SVG_NOTIFICATION="$SVG_DIR/white_transparent_icon.svg"

# Density map and sizes (launcher: mdpi=48 ... xxxhdpi=192; notification: mdpi=24 ... xxxhdpi=96)
declare -A LAUNCHER_SIZES=( [mdpi]=48 [hdpi]=72 [xhdpi]=96 [xxhdpi]=144 [xxxhdpi]=192 )
declare -A NOTIF_SIZES=( [mdpi]=24 [hdpi]=36 [xhdpi]=48 [xxhdpi]=72 [xxxhdpi]=96 )

# Find an SVG->PNG converter
if command -v rsvg-convert >/dev/null 2>&1; then
  CONVERTER="rsvg-convert"
elif command -v inkscape >/dev/null 2>&1; then
  CONVERTER="inkscape"
elif command -v convert >/dev/null 2>&1; then
  CONVERTER="convert"
else
  echo "No SVG converter found. Install 'rsvg-convert' (librsvg), 'inkscape', or ImageMagick." >&2
  exit 2
fi

if command -v magick >/dev/null 2>&1; then
  IMAGE_COMPOSER="magick"
elif command -v convert >/dev/null 2>&1; then
  IMAGE_COMPOSER="convert"
else
  echo "No PNG compositor found. Install ImageMagick." >&2
  exit 2
fi

svg_to_png() {
  local svg="$1" size="$2" out="$3"
  if [ "$CONVERTER" = "rsvg-convert" ]; then
    rsvg-convert -w "$size" "$svg" -o "$out"
  elif [ "$CONVERTER" = "inkscape" ]; then
    inkscape "$svg" --export-width="$size" -o "$out"
  else
    convert "$svg" -resize "${size}x${size}" "$out"
  fi
}

mkdir -p "$ANDROID_RES"

# --- Launcher icons (background plus transparent foreground, one per density) ---
for d in mdpi hdpi xhdpi xxhdpi xxxhdpi; do
  DIR="$ANDROID_RES/mipmap-$d"
  FOREGROUND_DIR="$ANDROID_RES/drawable-$d"
  mkdir -p "$DIR" "$FOREGROUND_DIR"
  size=${LAUNCHER_SIZES[$d]}
  echo "Launcher $d (${size}px)"
  svg_to_png "$SVG_BACKGROUND" "$size" "$DIR/ic_launcher.png"
  svg_to_png "$SVG_FOREGROUND" "$size" "$FOREGROUND_DIR/ic_launcher_foreground.png"
  if [ "$IMAGE_COMPOSER" = "magick" ]; then
    magick "$DIR/ic_launcher.png" "$FOREGROUND_DIR/ic_launcher_foreground.png" \
      -compose over -composite "$DIR/ic_launcher.png"
  else
    convert "$DIR/ic_launcher.png" "$FOREGROUND_DIR/ic_launcher_foreground.png" \
      -compose over -composite "$DIR/ic_launcher.png"
  fi
  cp -f "$DIR/ic_launcher.png" "$DIR/ic_launcher_round.png"
done

# --- Notification icons (white glyph on transparent background, one per density) ---
for d in mdpi hdpi xhdpi xxhdpi xxxhdpi; do
  DIR="$ANDROID_RES/drawable-$d"
  mkdir -p "$DIR"
  size=${NOTIF_SIZES[$d]}
  echo "Notification $d (${size}px)"
  svg_to_png "$SVG_NOTIFICATION" "$size" "$DIR/ic_stat_notify.png"
done

# --- Adaptive launcher background PNG ---
mkdir -p "$ANDROID_RES/drawable"
echo "Adaptive background drawable (432px)"
svg_to_png "$SVG_BACKGROUND" 432 \
  "$ANDROID_RES/drawable/ic_launcher_background.png"

# Keep density-specific background copies in sync for older resource resolution paths.
for d in mdpi hdpi xhdpi xxhdpi xxxhdpi; do
  echo "Adaptive background $d (432px)"
  svg_to_png "$SVG_BACKGROUND" 432 \
    "$ANDROID_RES/drawable-$d/ic_launcher_background.png"
done

echo "Icon generation finished. Review files under $ANDROID_RES/* and commit them."
