# 🧠 LEARNING_SYSTEM.md — Memoria de Aprendizaje y Prevención de Errores (Lazymark)

> **Ubicación:** `/Users/drizzy/Documents/proyectos/lazymark/LEARNING_SYSTEM.md`  
> **Objetivo:** Registro de lecciones de arquitectura, bugs resueltos y casos de prueba añadidos al harness.  

---

## 1. Registro de Lecciones Técnicas Iniciales

### Lección 01: La regla de nombres `*_test.go` en Go
* **Síntoma:** Al compilar el paquete `mouse`, `go vet` arrojaba: `no non-test Go files in .../internal/ui/mouse`.
* **Causa Raíz:** El archivo se nombró `hit_test.go`. En el toolchain de Go, cualquier archivo que termine con `_test.go` se considera automáticamente un archivo de test unitario.
* **Solución:** Renombrado a `zones.go` y su prueba a `zones_test.go`.
* **Mecanismo Preventivo:** Validación estricta en el harness contra advertencias de paquetes Go.

### Lección 02: Limpieza de Imágenes Kitty (Garbage Collection Gráfico)
* **Síntoma:** En terminales con soporte Kitty Graphics, al pasar de una nota que contiene imágenes a una nota de texto plano, la imagen previa puede quedar superpuesta como imagen fantasma.
* **Causa Raíz:** El protocolo Kitty retiene las imágenes en el búfer gráfico hasta recibir una orden explícita de eliminación.
* **Solución:** Implementación de `ClearAllCommand()` (`\x1b_Ga=d\x1b\\`) cada vez que se cambia de nota o se sale de la aplicación.

### Lección 03: Control de Procesos Externos sin Bloqueo de Terminal
* **Síntoma:** Al invocar editores como `micro` o `nano` directamente con `exec.Command().Run()`, la terminal quedaba congelada en modo raw de Bubble Tea.
* **Solución:** Utilizar obligatoriamente `tea.ExecProcess(c, callback)`. Este helper de Bubble Tea suspende el ciclo de eventos, cede la entrada/salida completa a `micro` y restaura el estado limpio al volver.

---

## 2. Protocolo de Extensión de Regresiones
Cualquier nuevo error reportado debe seguir el ciclo:
1. Aislar la causa raíz.
2. Añadir la prueba que reproduzca el fallo en `harness/validate_project.py` o en los tests unitarios `*_test.go`.
3. Implementar el parche.
4. Documentar aquí el aprendizaje.
