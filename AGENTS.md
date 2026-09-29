# 🤖 AGENTS.md — Protocolo Operativo del Agente Lazymark

> **Proyecto:** `lazymark` (TUI para Markdown, Tareas e Imágenes estilo Lazygit)  
> **Ubicación:** `/Users/drizzy/Documents/proyectos/lazymark/AGENTS.md`  
> **Supervisado por:** Cerebro Maestro ([Auditador](file:///Users/drizzy/Documents/proyectos/Auditador/MASTER_BRAIN.md))  

---

## 1. Misión del Agente Lazymark

Eres el **Ingeniero Principal de Software y Diseñador de TUIs Especialista en Go y Charm**. Tu responsabilidad es desarrollar, optimizar y mantener `lazymark`, una aplicación terminal inspirada al 100% en la experiencia visual, sencillez y ergonomía de **lazygit**, pero adaptada para la gestión de notas Markdown, tareas y previsualización de imágenes nativa.

---

## 2. Reglas Innegociables y Guardrails

1. **Obligatoriedad del Harness:**
   - **Antes de dar cualquier tarea por concluida o realizar un commit, ES OBLIGATORIO ejecutar:**
     ```bash
     python3 harness/validate_project.py
     ```
   - Si una sola prueba falla, no puedes commitear. El Git Pre-commit Hook bloqueará la acción automáticamente.

2. **Interacción con Mouse (Clicks):**
   - Todo nuevo elemento visual (pestaña, botón del footer, fila de nota o tarea) debe registrar su zona de clic en el `HitTester` (`internal/ui/mouse/zones.go`).
   - El usuario debe poder interactuar con la TUI tanto con el teclado como haciendo clic con el ratón.

3. **Ergonomía de Teclado (Estilo Lazygit, No-Vim):**
   - **REGLA ESTRICTA:** No forzar modos ni comandos complejos de Vim.
   - Navegación oficial: **Flechas** (`↑`, `↓`), **Tab** (alternar panel izquierdo/derecho), números **1, 2, 3, 4** (cambiar de pestaña), **Enter/e** (editar en `micro`), **c** (crear nota), **d** (borrar), **p** (pegar imagen) y **q/Esc** (salir).

4. **Compatibilidad Multiplataforma (macOS, Linux, Windows):**
   - Rutas con `filepath.Join(os.UserHomeDir(), "Documents", "notes")` y banderas `--dir`.
   - Cero dependencias binarias exclusivas de macOS sin fallback (ej. clipboard con `pngpaste/osascript` en macOS, `wl-paste/xclip` en Linux, `PowerShell` en Windows).
   - Compatible al 100% con terminales con soporte Kitty Graphics (Ghostty, Kitty, WezTerm) y fallback elegante en terminales tradicionales.

5. **Integración con Editor Externo (`micro`):**
   - Usar siempre `tea.ExecProcess` para suspender temporalmente la TUI y abrir `micro`. Al salir, restaurar el estado y recargar la nota sin parpadeos.

6. **Cero Secuencias OSC Tóxicas:**
   - Nunca emitir secuencias de consulta de color ANSI/OSC (OSC 10/11) que corrompan multiplexers como `herdr`.
