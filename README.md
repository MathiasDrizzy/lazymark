# 📝 Lazymark — Terminal Markdown Powerhouse (Estilo Lazygit)

> **Una TUI nativa, veloz y elegante para la gestión de notas Markdown, tareas y previsualización de imágenes, inspirada al 100% en la ergonomía de lazygit.**  
> Desarrollada en **Go** con el ecosistema **Charm** (`bubbletea`, `lipgloss`, `bubbles`, `glamour`).

---

## ✨ Características Principales

* 🖱️ **Soporte Completo para Ratón (Clicks):** Selecciona notas, tareas o imágenes, cambia de pestaña y activa botones del footer haciendo clic con el mouse.
* ⌨️ **Ergonomía Dual (Lazygit + Vim):**
  - **Navegación Lazygit:** Flechas (`↑`, `↓`), `Tab` (alternar panel izquierdo/derecho), números `1, 2, 3, 4` (pestañas) y atajos rápidos.
  - **Compatibilidad Vim:** Atajos familiares `j` / `k` (bajar/subir), `h` / `l` (paneles), `g` / `G` (inicio/fin).
* 📑 **4 Pestañas Especializadas:**
  1. **[1] Notas:** Lista con orden cronológico y renderizado Markdown con Glamour.
  2. **[2] Categorías/Tags:** Agrupación y conteo por hashtags (`#tag`), con vista previa de notas filtradas.
  3. **[3] Tareas:** Lista agregada de checkboxes Markdown (`- [ ]` / `- [x]`) con filtro interactivo (`f`: Todas / Pendientes / Completadas).
  4. **[4] Imágenes/Adjuntos:** Galería de capturas y diagramas incrustados.
* 🎨 **7 Temas de Color Dinámicos (`t` / `--theme`):**
  - `catppuccin-mocha` (por defecto)
  - `catppuccin-latte` (claro)
  - `catppuccin-frappe`
  - `catppuccin-macchiato`
  - `tokyo-night`
  - `gruvbox-dark`
  - `nord`
* 🖼️ **Protocolo Gráfico Kitty Nativo:** Previsualización de imágenes adjuntas (`.png`, `.jpg`, etc.) dentro de la terminal en emuladores compatibles (**Ghostty**, **Kitty**, **WezTerm**), con fallback ASCII elegante en terminales tradicionales.
* 📋 **Pegado Directo desde el Portapapeles:** Presiona `p` para pegar una captura del portapapeles directamente en la nota activa.
* 📝 **Integración con Editor Externo (`micro` / `$EDITOR`):** Presiona `Enter` o `e` para suspender temporalmente la TUI y abrir tu editor. Al salir, la sesión se restaura al instante.
* 🌍 **Compatibilidad Multiplataforma Real:** macOS, Linux (X11 con `xclip` o Wayland con `wl-paste`) y Windows (PowerShell nativo).
* 🛡️ **Arnés de Calidad Integrado:** Pruebas unitarias con detector de condiciones de carrera (`-race`), análisis estático (`go vet`), y verificación de escape codes sin interferencia con multiplexers.

---

## 🗂️ Interfaz Visual

```
┌────────────────────────────────────────────────────────────────────────┐
│ [1] Notas   [2] Categorías/Tags   [3] Tareas   [4] Imágenes/Adjuntos   │
├─────────────────────────┬──────────────────────────────────────────────┤
│ ❯ Mi Primera Nota 29 Sep│ # Mi Primera Nota                            │
│   Arquitectura AWS 28 Sep│                                              │
│   Ideas Homelab   27 Sep│ Notas y especificaciones en Markdown...      │
│                         │                                              │
│                         │ - [ ] Tarea pendiente                        │
│                         │ - [x] Tarea completada                       │
│                         │                                              │
│                         │ ![Diagrama](assets/arch.png)                 │
├─────────────────────────┴──────────────────────────────────────────────┤
│ [c] Nueva   [e/Enter] Editar en micro   [d] Borrar   [p] Pegar imagen  │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 🚀 Instalación y Distribución

### Vía Homebrew (macOS / Linux)
```bash
brew install MathiasDrizzy/tap/lazymark
```

### Vía Script de Instalación Rápida (`curl`)
```bash
curl -fsSL https://raw.githubusercontent.com/MathiasDrizzy/lazymark/main/scripts/install.sh | bash
```

### Vía Go (desde fuentes)
```bash
go install github.com/MathiasDrizzy/lazymark/cmd/lazymark@latest
```

### Compilación Manual
```bash
git clone https://github.com/MathiasDrizzy/lazymark.git
cd lazymark
go build -o bin/lazymark ./cmd/lazymark
./bin/lazymark
```

---

## ⌨️ Guía de Atajos de Teclado

| Atajo | Acción | Contexto |
|---|---|---|
| `1`, `2`, `3`, `4` | Cambiar a pestaña [Notas, Tags, Tareas, Galería] | Global |
| `↑` / `↓` ó `k` / `j` | Navegar hacia arriba / abajo | Listas |
| `←` / `→` ó `h` / `l` | Alternar entre panel de lista y vista previa | Global |
| `Tab` / `Shift+Tab` | Alternar entre paneles | Global |
| `g` / `G` | Ir al primer / último ítem | Listas |
| `Enter` / `e` | Abrir nota en `$EDITOR` (`micro`) | Global |
| `c` | Crear nueva nota rápida | Notas |
| `d` | Eliminar nota seleccionada | Notas |
| `p` | Pegar imagen del portapapeles | Notas / Galería |
| `f` | Alternar filtro (Todas → Pendientes → Completadas) | Tareas |
| `t` | Ciclar tema de colores dinámicamente | Global |
| `q` / `Esc` / `Ctrl+C` | Salir limpiando imágenes Kitty de la terminal | Global |
| **Clic izquierdo** | Seleccionar ítems, pestañas y botones del footer | Todo con ratón |

---

## ⚙️ Opciones de Línea de Comandos

```bash
lazymark [opciones]

Opciones:
  --dir string       Ruta al directorio de notas (por defecto: ~/Documents/notes)
  --theme string     Tema inicial (catppuccin-mocha, nord, tokyo-night, gruvbox-dark, etc.)
  --no-mouse         Desactiva la interacción con ratón y clics
  --version, -v      Muestra la versión de lazymark y sale
```

---

## 🧪 Arnés de Calidad (Quality Harness)

Para validar la integridad del proyecto antes de publicar o contribuir:
```bash
python3 harness/validate_project.py
```
El arnés verifica automáticamente:
1. `go vet ./...` (análisis estático).
2. `go test -race ./...` (pruebas unitarias concurrentes sin condiciones de carrera).
3. `go build` (compilación estática del binario).
4. Smoke tests de CLI (`--version`, `--help`).
5. Detección de secuencias OSC tóxicas para multiplexers.
