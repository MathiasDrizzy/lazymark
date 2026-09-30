#!/usr/bin/env python3
"""
visual_qa.py — Validador automatizado de control de calidad visual para Lazymark TUI.
Verifica que las vistas renderizadas respeten estrictamente las dimensiones del terminal,
no desborden líneas, contengan bordes íntegros y exporta capturas SVG/HTML con 'rich'.
"""

import os
import re
import sys
from rich.console import Console
from rich.text import Text

ANSI_ESCAPE = re.compile(r'\x1b(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')

def strip_ansi(text):
    return ANSI_ESCAPE.sub('', text)

def validate_snapshot(ans_path, expected_w, expected_h, export_svg=True):
    if not os.path.exists(ans_path):
        print(f"❌ [FAIL] Archivo no encontrado: {ans_path}")
        return False

    with open(ans_path, "rb") as f:
        raw_bytes = f.read()

    ansi_content = raw_bytes.decode("utf-8", errors="replace")
    clean_content = strip_ansi(ansi_content)
    lines = clean_content.splitlines()

    errors = []

    # 1. Verificar número de líneas verticales
    if len(lines) > expected_h:
        errors.append(f"Desbordamiento vertical: {len(lines)} líneas > máximo permitido {expected_h}")

    # 2. Verificar ancho horizontal por línea
    max_line_len = 0
    for idx, line in enumerate(lines):
        line_len = len(line)
        if line_len > max_line_len:
            max_line_len = line_len
        if line_len > expected_w:
            errors.append(f"Línea {idx + 1} excede ancho: {line_len} cols > {expected_w} cols (contenido: {repr(line[:40])}...)")

    # 3. Verificar presencia de bordes básicos
    has_top_border = any('╭' in l or '┌' in l for l in lines)
    has_bottom_border = any('╰' in l or '└' in l for l in lines)
    if not has_top_border or not has_bottom_border:
        errors.append("Bordes de panel incompletos o ausentes en el snapshot")

    # 4. Exportar SVG y HTML con rich
    if export_svg:
        base_name = os.path.splitext(ans_path)[0]
        svg_file = f"{base_name}.svg"
        html_file = f"{base_name}.html"
        console = Console(record=True, width=expected_w, height=expected_h)
        rich_text = Text.from_ansi(ansi_content)
        console.print(rich_text)
        console.save_svg(svg_file, title=f"Lazymark TUI ({expected_w}x{expected_h})")
        console.save_html(html_file)

    if errors:
        print(f"❌ [FAIL] Snapshot {os.path.basename(ans_path)} ({expected_w}x{expected_h}):")
        for err in errors:
            print(f"   • {err}")
        return False
    else:
        print(f"✅ [PASS] Snapshot {os.path.basename(ans_path)} ({expected_w}x{expected_h}) | Líneas: {len(lines)}/{expected_h} | Ancho máx: {max_line_len}/{expected_w}")
        return True

def main():
    base_dir = os.path.dirname(os.path.abspath(__file__))
    test_cases = [
        ("snapshot_compact_80x24.ans", 80, 24),
        ("snapshot_standard_120x35.ans", 120, 35),
        ("snapshot_fullscreen_160x45.ans", 160, 45),
    ]

    all_pass = True
    print("\n=================================================================")
    print("      LAZYMARK VISUAL QA HARNESS — VALIDACIÓN DE PANTALLA        ")
    print("=================================================================\n")

    for filename, w, h in test_cases:
        path = os.path.join(base_dir, filename)
        if not validate_snapshot(path, w, h):
            all_pass = False

    print("\n-----------------------------------------------------------------")
    if all_pass:
        print("🎉 TODOS LOS SNAPSHOTS VISUALES CUMPLEN CON LA GEOMETRÍA PERFECTA")
        print("=================================================================\n")
        sys.exit(0)
    else:
        print("⚠️ SE DETECTARON ERRORES DE GEOMETRÍA O DESBORDAMIENTO VISUAL")
        print("=================================================================\n")
        sys.exit(1)

if __name__ == "__main__":
    main()
