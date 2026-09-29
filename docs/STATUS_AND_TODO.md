# 📋 Lazymark — Project Status & Pending Roadmap

> Comprehensive tracking of implemented features, active pending items, and distribution roadmap for the `lazymark` TUI.

---

## ✅ 1. Completed Features (Shipped in v0.1.0)

### 🗂️ Core 4-Tab Interface
- [x] **[1] Notes View:** Chronological note list, Glamour-powered Markdown preview panel, note deletion (`d`), quick template creation (`c`).
- [x] **[2] Categories / Tags View:** Automatic `#tag` extraction from note content, tag counter badges, filtered note preview panel.
- [x] **[3] Tasks View:** Aggregate checklist extractor (`- [ ]` / `- [x]`), cyclic filter (`f`: All → Pending → Completed), parent note reference, line number indicator, detailed task preview.
- [x] **[4] Image Gallery View:** Markdown image reference extractor, file-type icons, native Kitty Graphics preview with graceful ASCII fallback for standard terminals.

### ⌨️ Navigation & Ergonomics
- [x] **Lazygit Navigation:** Arrows (`↑`/`↓`), `Tab`/`Shift+Tab` panel toggle, numbers `1..4` for instant tab switching.
- [x] **Vim Keybindings:** `j`/`k` (scroll), `h`/`l` (panel toggle), `g`/`G` (top/bottom), `Esc` (exit).
- [x] **Full Mouse Support:** Internal geometric `HitTester` registering clickable zones for tabs, list rows, task items, gallery entries, and footer buttons.

### 🎨 Themes & Customization
- [x] **7 Hot-Swappable Themes:** `catppuccin-mocha` (default), `catppuccin-latte`, `catppuccin-frappe`, `catppuccin-macchiato`, `tokyo-night`, `gruvbox-dark`, and `nord`.
- [x] **Runtime Theme Cycling:** Press `t` to cycle themes on the fly.
- [x] **CLI Flag:** Start directly with any theme: `lazymark --theme nord`.

### 🛠️ Editor & Clipboard Integration
- [x] **Editor Suspension:** `tea.ExecProcess` suspends the TUI cleanly to open `micro` (or `$EDITOR`) without flickering or raw terminal corruption.
- [x] **Clipboard Image Pasting (`p`):** Cross-platform extraction:
  - macOS: `pngpaste` with AppleScript native fallback.
  - Linux: Wayland (`wl-paste`) and X11 (`xclip`).
  - Windows: PowerShell native clipboard drawing export.
- [x] **Cross-Platform Compatibility:** Tested for macOS, Linux, and Windows path separators (`filepath.ToSlash` for Markdown URLs, sanitization of Windows-prohibited characters).

---

## 📌 2. Pending Items — Community Distribution (Priority 1)

These items are required to allow users worldwide to install `lazymark` with single-command ease:

### 🍺 Homebrew Tap Setup
- [ ] **Create Tap Repository:** Create `https://github.com/MathiasDrizzy/homebrew-tap`.
- [ ] **Automate Formula Generation:** Link with `.goreleaser.yaml` so every tagged release (`git tag v0.1.0 && git push --tags`) automatically updates the `lazymark.rb` Homebrew formula.
- [ ] **User Command:**
  ```bash
  brew install MathiasDrizzy/tap/lazymark
  ```

### ⚡ One-Liner Curl Installer (`scripts/install.sh`)
- [ ] **First GitHub Release Tag:** Publish `v0.1.0` release so GitHub assets (`lazymark_darwin_arm64.tar.gz`, etc.) exist.
- [ ] **Test Script:** Verify `curl -fsSL https://raw.githubusercontent.com/MathiasDrizzy/lazymark/main/scripts/install.sh | bash` downloads and installs seamlessly on fresh systems.

### 🪟 Windows & Linux Package Managers
- [ ] **Scoop Bucket:** Create Scoop manifest for Windows users (`scoop bucket add ... && scoop install lazymark`).
- [ ] **Winget Submission:** Publish manifest to Microsoft Windows Package Manager Community Repo.
- [ ] **Arch Linux AUR:** Create `lazymark-bin` PKGBUILD on AUR.

---

## 🚀 3. Pending Items — Feature Roadmap (v0.2.0+)

### 🔍 Interactive Fuzzy Search (`/`)
- [ ] Modal input dialog triggered by `/`.
- [ ] Fuzzy matching against note titles, tags, and body content.
- [ ] Live result highlighting and instant navigation.

### 🔘 In-TUI Task Toggle (`Space`)
- [ ] Pressing `Space` on any task in tab [3] toggles `- [ ]` ↔ `- [x]` directly in the corresponding file on disk without launching an external editor.

### 📜 Preview Scroll & Mouse Wheel
- [ ] Support vertical scrolling in the right preview pane with mouse wheel (`WheelUp`/`WheelDown`) and keyboard keys (`PageUp`/`PageDown`/`Ctrl+U`/`Ctrl+D`).

### 🔄 Background Git Synchronization (`--sync`)
- [ ] Optional `--sync` flag to commit and push changes silently to a user-configured remote Git repository upon note edits.
- [ ] File watcher (`fsnotify`) to detect external changes made to markdown files while the TUI is open.

---

## 📋 Summary Table

| Category | Item | Status | Target |
|---|---|---|---|
| **Core TUI** | 4 Tab Views (Notes, Tags, Tasks, Images) | ✅ Shipped | v0.1.0 |
| **Ergonomics** | Lazygit + Vim Dual Keybindings | ✅ Shipped | v0.1.0 |
| **Mouse Engine** | Geometric HitTester for all views | ✅ Shipped | v0.1.0 |
| **Themes** | 7 Color Palettes + Runtime Switching (`t`) | ✅ Shipped | v0.1.0 |
| **Terminal Graphics** | Kitty Graphics Protocol + ASCII fallback | ✅ Shipped | v0.1.0 |
| **Cross-Platform** | macOS + Linux (Wayland/X11) + Windows | ✅ Shipped | v0.1.0 |
| **Packaging** | GoReleaser + GitHub Actions CI Matrix | ✅ Configured | v0.1.0 |
| **Distribution** | GitHub Release v0.1.0 tag | 🟡 Pending | v0.1.0 |
| **Distribution** | Homebrew Tap (`MathiasDrizzy/tap`) | 🟡 Pending | v0.1.0 |
| **Distribution** | Curl Installer verification | 🟡 Pending | v0.1.0 |
| **Productivity** | Fuzzy Search (`/`) | 🔵 Planned | v0.2.0 |
| **Productivity** | Task Checkbox Toggle (`Space`) | 🔵 Planned | v0.2.0 |
| **Productivity** | Preview Viewport Scrolling | 🔵 Planned | v0.2.0 |
| **Sync** | Git Background Sync (`--sync`) | 🔵 Planned | v0.3.0 |
