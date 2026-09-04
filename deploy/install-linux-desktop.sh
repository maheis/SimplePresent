#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bundle_dir="${1:-${project_dir}/build/linux/x64/release/bundle}"
install_dir="${HOME}/.local/opt/simplepresent"
applications_dir="${HOME}/.local/share/applications"
icons_dir="${HOME}/.local/share/icons/hicolor/128x128/apps"
desktop_file="${applications_dir}/be.heister.simplepresent.desktop"

if [[ ! -x "${bundle_dir}/simplepresent" ]]; then
  printf 'Linux bundle nicht gefunden: %s\n' "${bundle_dir}" >&2
  printf 'Zuerst ausfuehren: flutter build linux --release\n' >&2
  exit 1
fi

install -d "${install_dir}" "${applications_dir}" "${icons_dir}"
cp -a "${bundle_dir}/." "${install_dir}/"
install -m 0644 \
  "${install_dir}/data/flutter_assets/assets/icons/color_transparent_icon.png" \
  "${icons_dir}/be.heister.simplepresent.png"

cat > "${desktop_file}" <<EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=SimplePresent
Comment=Aufgabenverwaltung fuer den aktuellen Tag
Exec=${install_dir}/simplepresent %U
Path=${install_dir}
Icon=be.heister.simplepresent
Terminal=false
Categories=Office;Utility;
StartupNotify=true
StartupWMClass=simplepresent
EOF

chmod 0644 "${desktop_file}"
update-desktop-database "${applications_dir}" >/dev/null 2>&1 || true

printf 'SimplePresent installiert: %s\n' "${desktop_file}"