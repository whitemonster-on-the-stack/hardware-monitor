#!/bin/bash
set -e

APP_NAME="OmniTop"
APP_DIR="AppDir"
OUTPUT="OmniTop-x86_64.AppImage"

# Ensure script is run from the project root
if [ ! -f "go.mod" ]; then
    echo "Error: Please run this script from the project root directory."
    exit 1
fi

# Build the binary first
echo "Building OmniTop..."
go build -o omnitop ./cmd/omnitop

# Clean up previous build
rm -rf "$APP_DIR" "$OUTPUT"

# Create AppDir structure
mkdir -p "$APP_DIR/usr/bin"
mkdir -p "$APP_DIR/usr/share/icons/hicolor/256x256/apps"
mkdir -p "$APP_DIR/usr/share/applications"
mkdir -p "$APP_DIR/usr/share/metainfo"

# Copy binary
cp omnitop "$APP_DIR/usr/bin/"

# Create desktop file at root of AppDir (REQUIRED by AppImage spec) AND in /usr/share/applications
cat <<DESKTOP > "$APP_DIR/$APP_NAME.desktop"
[Desktop Entry]
Name=$APP_NAME
Exec=omnitop
Icon=omnitop
Type=Application
Categories=Utility;System;Monitor;
Terminal=true
EOF
cp "$APP_DIR/$APP_NAME.desktop" "$APP_DIR/usr/share/applications/"

# Create icon at root of AppDir (REQUIRED by AppImage spec) AND in standard location
if [ ! -f "omnitop.png" ]; then
    echo "Error: Icon file omnitop.png not found in project root. Please add a valid PNG icon."
    exit 1
fi
cp "omnitop.png" "$APP_DIR/omnitop.png"
cp "omnitop.png" "$APP_DIR/usr/share/icons/hicolor/256x256/apps/omnitop.png"

# Create AppRun script
cat <<EOF > "$APP_DIR/AppRun"
#!/bin/bash
HERE="\$(dirname "\$(readlink -f "\${0}")")"
export PATH="\${HERE}/usr/bin:\${PATH}"
exec "\${HERE}/usr/bin/omnitop" "\$@"
EOF
chmod +x "$APP_DIR/AppRun"

# Download appimagetool if not present
TOOL="appimagetool-x86_64.AppImage"
if [ ! -f "$TOOL" ]; then
    echo "Downloading appimagetool..."
    wget -q https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage -O "$TOOL"
    chmod +x "$TOOL"
fi

# Build AppImage
echo "Building AppImage..."
# Use ARCH env var to help appimagetool
ARCH=x86_64 ./$TOOL "$APP_DIR" "$OUTPUT"

echo "Success! AppImage created: $OUTPUT"
