#!/usr/bin/env sh
set -e

# Lazymark Universal Installer Script
# Uso: curl -fsSL https://raw.githubusercontent.com/MathiasDrizzy/lazymark/main/scripts/install.sh | bash

REPO="MathiasDrizzy/lazymark"
BINARY_NAME="lazymark"

detect_os_arch() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$ARCH" in
        x86_64|amd64) ARCH="x86_64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) echo "Arquitectura no soportada: $ARCH"; exit 1 ;;
    esac

    case "$OS" in
        darwin) OS="darwin" ;;
        linux) OS="linux" ;;
        *) echo "Sistema operativo no soportado: $OS"; exit 1 ;;
    esac
}

install_binary() {
    detect_os_arch
    TARGET_DIR="${INSTALL_DIR:-/usr/local/bin}"

    echo "🔍 Detectado: $OS ($ARCH)"
    echo "⬇️  Descargando última versión de $BINARY_NAME..."

    LATEST_URL="https://github.com/$REPO/releases/latest/download/${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
    TEMP_DIR="$(mktemp -d)"

    if curl -fsSL "$LATEST_URL" -o "$TEMP_DIR/lazymark.tar.gz" 2>/dev/null; then
        tar -xzf "$TEMP_DIR/lazymark.tar.gz" -C "$TEMP_DIR"
        chmod +x "$TEMP_DIR/$BINARY_NAME"
        
        if [ -w "$TARGET_DIR" ]; then
            mv "$TEMP_DIR/$BINARY_NAME" "$TARGET_DIR/"
        else
            echo "🔐 Se requieren permisos de superusuario para instalar en $TARGET_DIR:"
            sudo mv "$TEMP_DIR/$BINARY_NAME" "$TARGET_DIR/"
        fi
        rm -rf "$TEMP_DIR"
        echo "✅ ¡$BINARY_NAME instalado con éxito en $TARGET_DIR/$BINARY_NAME!"
        echo "🚀 Ejecuta 'lazymark' para iniciar."
    else
        # Fallback a compilación mediante Go si está instalado
        if command -v go >/dev/null 2>&1; then
            echo "⚠️  No se encontró binario precompilado. Compilando con Go..."
            go install "github.com/$REPO/cmd/lazymark@latest"
            echo "✅ ¡$BINARY_NAME instalado con éxito mediante 'go install'!"
        else
            echo "❌ No se pudo descargar la release y Go no está instalado en el sistema."
            echo "👉 Visita https://github.com/$REPO/releases para descargas manuales."
            exit 1
        fi
    fi
}

install_binary
