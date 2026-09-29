# 📝 Lazymark — Terminal Markdown Powerhouse (Estilo Lazygit)

> **Una TUI nativa, veloz y elegante para la gestión de notas Markdown, tareas y previsualización de imágenes, inspirada al 100% en la ergonomía de lazygit.**  
> Desarrollada en **Go** con el ecosistema **Charm** (`bubbletea`, `lipgloss`, `bubbles`, `glamour`).

---

## ✨ Características Principales

* 🖱️ **Soporte Completo para Ratón (Clicks):** Selecciona notas, cambia de pestaña y ejecuta acciones en el footer directamente haciendo clic con el mouse.
* ⌨️ **Ergonomía Lazygit (No-Vim):** Navegación oficial con **Flechas** (`↑`, `↓`), **Tab** (alternar panel izquierdo/derecho), números **1, 2, 3, 4** (pestañas) y atajos intuitivos.
* 🖼️ **Protocolo Gráfico Kitty Nativo:** Previsualización de imágenes adjuntas (`.png`, `.jpg`) directamente dentro de la terminal en emuladores compatibles como **Ghostty**, Kitty y WezTerm.
* 📋 **Pegado de Imágenes desde el Portapapeles:** Presiona `p` para pegar una captura del portapapeles directamente en la nota activa.
* 📝 **Integración con Editor Externo (`micro`):** Presiona `Enter` o `e` para editar la nota en `micro` (o `$EDITOR`). Al salir, la TUI se reanuda al instante sin parpadeos.
* 🎨 **Estética Catppuccin Mocha:** Colores TrueColor armonizados (Peach, Mauve, Teal, Surface0).
* 🌍 **Multiplataforma:** Compatible con **macOS**, **Linux** y **Windows**.
* 🛡️ **Blindado con Harness:** Suite de calidad automatizada con pre-commit hook (`harness/validate_project.py`).

---

## 🗂️ Arquitectura de Interfaz

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

## 🚀 Compilación e Instalación

```bash
# Compilar binario
go build -o bin/lazymark ./cmd/lazymark

# Ejecutar
./bin/lazymark

# Especificar directorio de notas personalizado
./bin/lazymark --dir ~/Documents/mis-notas
```

---

## 🧪 Arnés de Calidad (Harness)

Antes de cada commit, ejecuta el arnés de verificación:
```bash
python3 harness/validate_project.py
```
El arnés ejecuta:
1. `go vet ./...` (análisis estático).
2. `go test -race ./...` (pruebas unitarias con detector de condiciones de carrera).
3. `go build` (compilación estática de producción).
4. Smoke tests (`--version`, `--help`).
5. Verificación de ausencia de secuencias OSC tóxicas para multiplexers (`herdr`).
