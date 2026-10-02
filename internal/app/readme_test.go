package app

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func readRepo(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(repoFile(rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestReadmeStructure (guía §2): el README sigue el orden estándar, con el GIF
// principal antes de la primera sección y como máximo 3 badges.
func TestReadmeStructure(t *testing.T) {
	text := readRepo(t, "README.md")
	order := []string{"## Why lazymark", "## What it does", "## Install", "## Quick start", "## Keys", "## Configuration", "## Compatibility", "## Contributing", "## License"}
	prev := -1
	for _, h := range order {
		i := strings.Index(text, "\n"+h+"\n")
		if i < 0 {
			t.Fatalf("falta la sección %q", h)
		}
		if i < prev {
			t.Errorf("la sección %q está fuera de orden", h)
		}
		prev = i
	}
	first := strings.Index(text, "\n## ")
	if g := strings.Index(text, "assets/readme/main.gif"); g < 0 || g > first {
		t.Error("el GIF principal debe verse antes de la primera sección")
	}
	if n := strings.Count(text, "[!["); n > 3 {
		t.Errorf("hay %d badges (máximo 3)", n)
	}
	for _, link := range []string{"[Install](#install)", "[Quick start](#quick-start)", "[Keys](#keys)", "[Configuration](#configuration)"} {
		if !strings.Contains(text, link) {
			t.Errorf("faltan los enlaces rápidos: %s", link)
		}
	}
	// una sección de funcionalidades con un GIF (o captura) por bloque
	what := text[strings.Index(text, "## What it does"):strings.Index(text, "## Install")]
	if blocks, media := strings.Count(what, "\n### "), strings.Count(what, "<img src=\"assets/readme/"); blocks < 4 || blocks != media {
		t.Errorf("funcionalidades: %d bloques y %d imágenes (4 o más, una por bloque)", blocks, media)
	}
}

// emoji devuelve cuántos emojis hay en una línea; ✓ y ✗ se admiten en las tablas.
func emojiCount(line string) int {
	n := 0
	for _, r := range line {
		switch {
		case r >= 0x1F000 && r <= 0x1FAFF, r >= 0x2B00 && r <= 0x2BFF, r == 0xFE0F, r == 0x200D:
			n++
		case r >= 0x2600 && r <= 0x27BF:
			if (r == 0x2713 || r == 0x2717) && strings.HasPrefix(strings.TrimSpace(line), "|") {
				continue
			}
			n++
		}
	}
	return n
}

// TestReadmeNoDecorativeEmoji (C2): ningún emoji decorativo (solo ✓ y ✗ en tablas).
func TestReadmeNoDecorativeEmoji(t *testing.T) {
	for _, f := range []string{"README.md", "docs/keybindings.md", "docs/configuration.md"} {
		for i, line := range strings.Split(readRepo(t, f), "\n") {
			if n := emojiCount(line); n > 0 {
				t.Errorf("%s:%d tiene %d emoji decorativo(s): %q", f, i+1, n, line)
			}
		}
	}
	if emojiCount("# Title 🚀") == 0 || emojiCount("| a | ✓ |") != 0 || emojiCount("✓ fuera de tabla") == 0 {
		t.Error("el detector de emojis no distingue lo que debe")
	}
}

var (
	imgTag   = regexp.MustCompile(`<img src="([^"]+)" alt="([^"]*)"`)
	mdLink   = regexp.MustCompile(`\]\(([^)#][^)]*)\)`)
	docImage = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
)

var (
	codeFence = regexp.MustCompile("(?s)```.*?```")
	codeSpan  = regexp.MustCompile("`[^`\n]*`")
)

// withoutCode quita los bloques y los fragmentos de código: lo que muestran
// (p. ej. la sintaxis ![](assets/…)) no es un enlace ni una imagen del documento.
func withoutCode(text string) string {
	return codeSpan.ReplaceAllString(codeFence.ReplaceAllString(text, ""), "")
}

// TestReadmeAssetsAndLinks (C3, C9): todo enlace e imagen relativos existen, cada
// imagen lleva alt, y cada GIF sale de un .tape versionado (el PNG está documentado).
func TestReadmeAssetsAndLinks(t *testing.T) {
	for _, f := range []string{"README.md", "docs/keybindings.md", "docs/configuration.md"} {
		text := withoutCode(readRepo(t, f))
		base := filepath.Dir(f)
		var targets []string
		for _, m := range imgTag.FindAllStringSubmatch(text, -1) {
			if strings.TrimSpace(m[2]) == "" {
				t.Errorf("%s: la imagen %s no tiene alt", f, m[1])
			}
			targets = append(targets, m[1])
		}
		for _, m := range docImage.FindAllStringSubmatch(text, -1) {
			if strings.TrimSpace(m[1]) == "" {
				t.Errorf("%s: la imagen %s no tiene alt", f, m[2])
			}
			targets = append(targets, m[2])
		}
		for _, m := range mdLink.FindAllStringSubmatch(text, -1) {
			targets = append(targets, m[1])
		}
		for _, tg := range targets {
			if strings.Contains(tg, "://") || strings.HasPrefix(tg, "mailto:") {
				continue
			}
			if _, err := os.Stat(repoFile(filepath.ToSlash(filepath.Join(base, tg)))); err != nil {
				t.Errorf("%s: el enlace relativo %q no existe", f, tg)
			}
		}
	}
	captures := readRepo(t, "assets/readme/CAPTURES.md")
	for _, m := range imgTag.FindAllStringSubmatch(readRepo(t, "README.md"), -1) {
		src := m[1]
		name := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
		switch filepath.Ext(src) {
		case ".gif":
			tape := readRepo(t, "assets/readme/tapes/"+name+".tape")
			if !strings.Contains(tape, "Output "+src) {
				t.Errorf("%s: su tape no escribe esa salida", src)
			}
		case ".png":
			if !strings.Contains(captures, filepath.Base(src)) {
				t.Errorf("%s: la captura no está documentada en assets/readme/CAPTURES.md", src)
			}
		}
	}
}

// TestReadmeWeights (C5): README < 500 KiB, GIF principal ≤ 5 MB y los demás ≤ 2 MB.
func TestReadmeWeights(t *testing.T) {
	if fi, _ := os.Stat(repoFile("README.md")); fi.Size() >= 500*1024 {
		t.Errorf("README.md pesa %d bytes", fi.Size())
	}
	files, _ := filepath.Glob(repoFile("assets/readme/*.gif"))
	files2, _ := filepath.Glob(repoFile("assets/readme/*.png"))
	for _, f := range append(files, files2...) {
		fi, _ := os.Stat(f)
		limit := int64(2 << 20)
		if filepath.Base(f) == "main.gif" {
			limit = 5 << 20
		}
		if fi.Size() > limit {
			t.Errorf("%s pesa %d bytes (máximo %d)", filepath.Base(f), fi.Size(), limit)
		}
	}
}

// TestDemoHomeHasNoPersonalData (C4): lo que viaja en assets/readme es inventado y
// está en inglés; ningún archivo, tampoco los binarios, trae rutas ni datos del autor.
func TestDemoHomeHasNoPersonalData(t *testing.T) {
	forbidden := []string{"/users/", "\\users\\", "drizzy", "mathias", "sonza", "gmail", "/var/folders", "/private/tmp"}
	root := repoFile("assets/readme")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		low := bytes.ToLower(data)
		for _, bad := range forbidden {
			if bytes.Contains(low, []byte(bad)) {
				t.Errorf("%s contiene %q", path, bad)
			}
		}
		if strings.Contains(path, "demo-home") && strings.HasSuffix(path, ".md") {
			if regexp.MustCompile(`[áéíóúñ¿¡]`).Match(data) {
				t.Errorf("%s: las notas de demo deben estar en inglés", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestInstallCommandMatchesModule: el comando `go install` del README apunta al módulo y al paquete reales.
func TestInstallCommandMatchesModule(t *testing.T) {
	mod := regexp.MustCompile(`(?m)^module (\S+)`).FindStringSubmatch(readRepo(t, "go.mod"))
	if mod == nil {
		t.Fatal("go.mod sin módulo")
	}
	want := "go install " + mod[1] + "/cmd/lazymark@latest"
	if !strings.Contains(readRepo(t, "README.md"), want) {
		t.Errorf("el README debe instalar con: %s", want)
	}
	if _, err := os.Stat(repoFile("cmd/lazymark/main.go")); err != nil {
		t.Error("el paquete cmd/lazymark no existe")
	}
	goVer := regexp.MustCompile(`(?m)^go (\S+)`).FindStringSubmatch(readRepo(t, "go.mod"))
	if goVer == nil || !strings.Contains(readRepo(t, "README.md"), "Go "+goVer[1]+" or newer") {
		t.Errorf("el README debe decir la versión de Go de go.mod (%v)", goVer)
	}
}
