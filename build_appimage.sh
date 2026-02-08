#!/bin/bash
set -e

APP_NAME="OmniTop"
APP_DIR="AppDir"
OUTPUT="OmniTop-x86_64.AppImage"

# Build the binary first
echo "Building OmniTop..."
go build -o omnitop ./cmd/omnitop

# Clean up previous build
rm -rf "$APP_DIR" "$OUTPUT"

# Create AppDir structure
mkdir -p "$APP_DIR/usr/bin"
mkdir -p "$APP_DIR/usr/share/icons/hicolor/256x256/apps"
mkdir -p "$APP_DIR/usr/share/applications"

# Copy binary
cp omnitop "$APP_DIR/usr/bin/"

# Create desktop file
cat <<DESKTOP > "$APP_DIR/usr/share/applications/$APP_NAME.desktop"
[Desktop Entry]
Name=$APP_NAME
Exec=omnitop
Icon=omnitop
Type=Application
Categories=Utility;System;Monitor;
Terminal=true
DESKTOP

# Create icon (dummy for now, replace with actual icon if available)
touch "$APP_DIR/usr/share/icons/hicolor/256x256/apps/omnitop.png"

# Create AppRun script
cat <<APPRUN > "$APP_DIR/AppRun"
#!/bin/bash
HERE="$(dirname "$(readlink -f "${0}")")"
export PATH="${HERE}/usr/bin:${PATH}"
exec "${HERE}/usr/bin/omnitop" "$@"
APPRUN
chmod +x "$APP_DIR/AppRun"

# Download appimagetool if not present
if [ ! -f "appimagetool-x86_64.AppImage" ]; then
    echo "Downloading appimagetool..."
    wget -q https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
    chmod +x appimagetool-x86_64.AppImage
fi

# Build AppImage
echo "Building AppImage..."
# Use ARCH env var to help appimagetool
ARCH=x86_64 ./appimagetool-x86_64.AppImage "$APP_DIR" "$OUTPUT"

echo "Success! AppImage created: $OUTPUT"
