#!/usr/bin/env python3
"""
Lazymark Test & Quality Harness
Ejecuta la suite de verificación exhaustiva del proyecto en Go:
- go vet ./... (análisis estático de sintaxis y buenas prácticas)
- go test -race ./... (pruebas unitarias y detección de race conditions)
- go build ./cmd/lazymark (compilación estática de binario)
- Smoke Test: validación de flags --version y --help
- Verificación de ausencia de secuencias OSC que rompan multiplexers (como herdr)
"""

import os
import subprocess
import sys

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
passed_checks = 0
failed_checks = 0

def log_pass(msg):
    global passed_checks
    passed_checks += 1
    print(f"  [PASS] {msg}")

def log_fail(msg, error=None):
    global failed_checks
    failed_checks += 1
    print(f"  [FAIL] {msg}")
    if error:
        print(f"         Detalle: {error.strip()}")

def test_go_vet():
    print("\n🔍 Ejecutando Go Vet (Análisis Estático)...")
    res = subprocess.run(["go", "vet", "./..."], cwd=REPO_ROOT, capture_output=True, text=True)
    if res.returncode == 0:
        log_pass("go vet pasó sin errores ni advertencias")
    else:
        log_fail("go vet detectó problemas", res.stderr)

def test_go_tests_race():
    print("\n🏃 Ejecutando Pruebas Unitarias con Race Detector (-race)...")
    res = subprocess.run(["go", "test", "-race", "./..."], cwd=REPO_ROOT, capture_output=True, text=True)
    if res.returncode == 0:
        log_pass("Todas las pruebas unitarias pasaron sin race conditions")
    else:
        log_fail("Fallo en suite de pruebas unitarias", res.stderr + "\n" + res.stdout)

def get_bin_target():
    bin_name = "lazymark.exe" if sys.platform == "win32" else "lazymark"
    return os.path.join(REPO_ROOT, "bin", bin_name)

def test_go_build():
    print("\n🔨 Compilando Binario Lazymark...")
    bin_target = get_bin_target()
    os.makedirs(os.path.dirname(bin_target), exist_ok=True)
    res = subprocess.run(["go", "build", "-o", bin_target, "./cmd/lazymark"], cwd=REPO_ROOT, capture_output=True, text=True)
    if res.returncode == 0:
        log_pass(f"Binario compilado con éxito: {bin_target}")
    else:
        log_fail("Error durante la compilación de lazymark", res.stderr)

def test_smoke_cli():
    print("\n💨 Ejecutando Smoke Tests de CLI (--version, --help)...")
    bin_target = get_bin_target()
    if not os.path.exists(bin_target):
        log_fail("Binario no encontrado para smoke test")
        return

    # Probar --version
    res_ver = subprocess.run([bin_target, "--version"], capture_output=True, text=True)
    if res_ver.returncode == 0 and "lazymark v" in res_ver.stdout:
        log_pass(f"--version respondió correctamente: {res_ver.stdout.strip()}")
    else:
        log_fail("--version falló o formato incorrecto", res_ver.stderr)

    # Probar --help
    res_help = subprocess.run([bin_target, "--help"], capture_output=True, text=True)
    if "Uso:" in res_help.stderr or "Uso:" in res_help.stdout:
        log_pass("--help respondió con la guía de atajos de lazygit")
    else:
        log_fail("--help falló al emitir instrucciones")

def test_herdr_compatibility():
    print("\n🛡️ Verificando Compatibilidad con herdr y Multiplexers...")
    # Verificar que el código fuente no incluya secuencias OSC 10 / OSC 11 de consulta de color
    # que causaron el bug en shiki
    dangerous_patterns = [r"\x1b]10;?", r"\x1b]11;?", "OSC 10", "OSC 11"]
    found = False
    for root, _, files in os.walk(os.path.join(REPO_ROOT, "internal")):
        for f in files:
            if f.endswith(".go"):
                path = os.path.join(root, f)
                with open(path, "r", encoding="utf-8") as file:
                    content = file.read()
                    for p in dangerous_patterns:
                        if p in content:
                            log_fail(f"Patrón peligroso '{p}' encontrado en {f}")
                            found = True
    if not found:
        log_pass("Cero emisiones de consultas OSC que puedan confundir a herdr o terminales")

def main():
    print("=" * 65)
    print("       LAZYMARK QUALITY HARNESS — VERIFICACIÓN CONTINUA")
    print("=" * 65)

    test_go_vet()
    test_go_tests_race()
    test_go_build()
    test_smoke_cli()
    test_herdr_compatibility()

    print("\n" + "=" * 65)
    print(f"RESUMEN: {passed_checks} Pruebas Exitosas | {failed_checks} Fallos")
    print("=" * 65)

    if failed_checks > 0:
        print("\n❌ EL HARNESS HA DETECTADO REGRESIONES O FALLOS.")
        sys.exit(1)
    else:
        print("\n✅ TODAS LAS PRUEBAS PASARON. BINARIO 100% ESTABLE.")
        sys.exit(0)

if __name__ == "__main__":
    main()
