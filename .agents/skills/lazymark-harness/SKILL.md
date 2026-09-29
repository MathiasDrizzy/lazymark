---
name: lazymark-harness
description: Ejecuta la suite completa de calidad, go vet, pruebas con race detector, compilación y smoke test para lazymark.
---

# Lazymark Quality Harness Skill

Usa esta skill cada vez que realices modificaciones en el código fuente de `lazymark`.

## Instrucciones

1. Ejecuta el harness desde la raíz del repositorio:
   ```bash
   python3 harness/validate_project.py
   ```

2. Comprueba que las 6 pruebas pasen (`[PASS]`).
3. Para ejecutar las pruebas unitarias directamente:
   ```bash
   go test -v -race ./...
   ```
4. Para compilar el binario local:
   ```bash
   go build -o bin/lazymark ./cmd/lazymark
   ```
