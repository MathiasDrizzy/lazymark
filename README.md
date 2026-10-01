# 📝 Lazymark — Terminal Markdown Powerhouse

<p align="center">
  <strong>A fast, ergonomic terminal UI for managing Markdown notes, aggregate tasks, and inline images — inspired 100% by the legendary experience of <a href="https://github.com/jesseduffield/lazygit">lazygit</a>.</strong>
</p>

<p align="center">
  <a href="https://github.com/MathiasDrizzy/lazymark/releases"><img src="https://img.shields.io/github/v/release/MathiasDrizzy/lazymark?color=fab387&style=flat-square" alt="Release"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.27%2B-blue?style=flat-square&logo=go" alt="Go Version"></a>
  <a href="https://github.com/MathiasDrizzy/lazymark/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License"></a>
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-cba6f7?style=flat-square" alt="Platform">
  <img src="https://img.shields.io/badge/terminal-Ghostty%20%7C%20WezTerm%20%7C%20Kitty-94e2d5?style=flat-square" alt="Terminals">
</p>

---

## 📸 The Interface

```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ [1] Notes        [2] Tags / Categories        [3] Tasks                                │
├───────────────────────────────┬────────────────────────────────────────────────────────┤
│ ❯ 📝 System Architecture      │ # System Architecture                                  │
│   📝 Weekly Review            │                                                        │
│   📝 Project Roadmaps         │ Comprehensive notes rendered with syntax highlighting. │
│   📝 Quick Scratchpad         │                                                        │
│                               │ - [ ] Ship v0.1.0 release to Homebrew                  │
│                               │ - [x] Implement Kitty Graphics protocol native support │
│                               │                                                        │
│                               │ ![Diagram](assets/diagram.png)                         │
├───────────────────────────────┴────────────────────────────────────────────────────────┤
│ [c] New   [e/Enter] Edit ($EDITOR)   [d] Delete   [t] Theme   [q] Quit                   │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## ✨ Features at a Glance

* 🖱️ **Full Mouse & Click Interactivity:** Click on any note row, task checkbox, top tab header, or footer action button.
* ⌨️ **Ergonomic Dual Navigation (Lazygit + Vim):**
  - **Lazygit Style:** Arrow keys (`↑`/`↓`), `Tab` / `Shift+Tab` to switch panels, numeric keys `1..3` to switch tabs.
  - **Vim Familiarity:** `j` / `k` (up/down), `h` / `l` (panels), `g` / `G` (top/bottom), `Esc` (exit).
* 📑 **3 Dedicated Panels:**
  1. **[1] Notes:** Chronological listing, formatted timestamps, and Glamour-powered Markdown preview.
  2. **[2] Tags / Categories:** Automatic extraction of `#hashtags`, note counters, and filtered tag previews.
  3. **[3] Tasks:** Global aggregation of checklist items (`- [ ]` / `- [x]`) across all notes, with cyclic filtering (`f`: All → Pending → Completed).
* 🖼️ **Native Kitty Graphics Protocol:** Inline image rendering for **Ghostty**, **Kitty**, and **WezTerm** with zero flickering and clean memory garbage collection. Includes an elegant ASCII fallback for traditional terminals.
* 📝 **Seamless External Editor Suspension:** Press `Enter` or `e` to suspend `lazymark` and open your favorite terminal editor (`micro`, `nvim`, `helix`, or `$EDITOR`). Your session resumes instantly upon exit.
* 🎨 **7 Hot-Swappable Color Themes:** Press `t` to cycle through `catppuccin-mocha`, `catppuccin-latte`, `catppuccin-frappe`, `catppuccin-macchiato`, `tokyo-night`, `gruvbox-dark`, and `nord` in real-time.
* 🌍 **True Cross-Platform Support:** Native support for macOS, Linux (Wayland `wl-paste` & X11 `xclip`), and Windows (PowerShell native).

---

## 🚀 Installation

### 1. Homebrew (macOS & Linux)
```bash
brew install MathiasDrizzy/tap/lazymark
```

### 2. Universal One-Liner (Curl & Bash)
```bash
curl -fsSL https://raw.githubusercontent.com/MathiasDrizzy/lazymark/main/scripts/install.sh | bash
```

### 3. Via Go
```bash
go install github.com/MathiasDrizzy/lazymark/cmd/lazymark@latest
```

### 4. Build from Source
```bash
git clone https://github.com/MathiasDrizzy/lazymark.git
cd lazymark
go build -o bin/lazymark ./cmd/lazymark
./bin/lazymark
```

---

## ⌨️ Keybindings Reference

| Shortcut | Action | Scope |
|---|---|---|
| `1`, `2`, `3` | Switch Tab ([1] Notes, [2] Tags, [3] Tasks) | Global |
| `↑` / `↓` or `k` / `j` | Navigate list items | Lists |
| `←` / `→` or `h` / `l` | Toggle between list and preview panel | Global |
| `Tab` / `Shift+Tab` | Toggle between list and preview panel | Global |
| `g` / `G` | Jump to first / last item | Lists |
| `Enter` / `e` | Open note in external editor (`micro` or `$EDITOR`) | Global |
| `c` | Create new note from quick template | Notes Tab |
| `d` | Delete selected note | Notes Tab |
| `f` | Cycle task filter (All → Pending → Completed) | Tasks Tab |
| `t` | Cycle color theme dynamically | Global |
| `q` / `Esc` / `Ctrl+C` | Quit and cleanly clear terminal graphics buffer | Global |
| **Left Click** | Select any note, task, tab, or footer button | Mouse |

---

## ⚙️ CLI Flags

```bash
lazymark [flags]

Flags:
  --dir string       Path to custom notes directory (default: ~/Documents/notes)
  --theme string     Initial color theme (mocha, latte, tokyo-night, gruvbox-dark, nord)
  --no-mouse         Disable mouse & click capture
  --version, -v      Print lazymark version and exit
  --help             Display usage guide
```

---

## 🎨 Themes Showcase

Lazymark includes 7 hand-tuned palettes out of the box. Press `t` to switch instantly or use `--theme`:

* `catppuccin-mocha` *(Default — Cozy, dark & balanced)*
* `catppuccin-latte` *(Clean, high-contrast light theme)*
* `catppuccin-frappe` *(Subtle cool dark tone)*
* `catppuccin-macchiato` *(Rich warm dark tone)*
* `tokyo-night` *(Modern deep blue aesthetic)*
* `gruvbox-dark` *(Classic retro warm palette)*
* `nord` *(Arctic, clean cold blue)*

---

## 📄 License

Lazymark is open-source software licensed under the **[MIT License](LICENSE)**.
