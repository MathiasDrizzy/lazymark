package image

import (
	"strings"
	"testing"
)

func TestKittyClient(t *testing.T) {
	client := &Client{Supported: true}

	cmd := client.RenderCommand("/path/to/test.png", 40, 20)
	if !strings.HasPrefix(cmd, "\x1b_Ga=T") {
		t.Errorf("Secuencia de escape inválida: %q", cmd)
	}

	clearCmd := client.ClearAllCommand()
	if clearCmd != "\x1b_Ga=d\x1b\\" {
		t.Errorf("Comando de limpieza inválido: %q", clearCmd)
	}

	unsupportedClient := &Client{Supported: false}
	fallback := unsupportedClient.RenderCommand("/path/to/test.png", 40, 20)
	if !strings.Contains(fallback, "Terminal sin soporte") {
		t.Errorf("Fallback no emitido en terminal no soportada: %s", fallback)
	}
}
