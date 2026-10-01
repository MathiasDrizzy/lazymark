package app

import (
	"errors"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/clipboard"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
)

// attachImage guarda una imagen en la carpeta assets/ de la nota que se está
// viendo (con save) e inserta su referencia `![](assets/…)`: al final de la
// nota o, con el foco en Tareas, debajo de la tarea seleccionada. Si no se
// puede insertar (p. ej. la nota cambió por fuera) no deja la imagen huérfana.
func (m *AppModel) attachImage(save func(noteDir, noteName string) (string, error)) {
	note := m.displayedNote()
	if note == nil || m.kanbanOn {
		m.c.setStatus("%s", i18n.T("Abre una nota para pegar una imagen", "Open a note to paste an image"))
		return
	}
	path, mod, dir := note.Path, note.ModTime, filepath.Dir(note.Path)
	ref, err := save(dir, filepath.Base(path))
	if err != nil {
		m.c.errStatus("No se pudo guardar la imagen", "Could not save the image", err)
		return
	}

	var task *views.FlatTask
	if m.focus == panelTasks {
		if t := m.tasks.current(); t != nil && t.NotePath == path {
			cp := *t
			task = &cp
		}
	}
	md := "![](" + ref + ")"
	if task != nil {
		err = m.c.store.InsertAfterLine(path, task.Line, md, mod)
	} else {
		err = m.c.store.AppendToNote(path, md, mod)
	}
	if err != nil {
		_ = os.Remove(filepath.Join(dir, filepath.FromSlash(ref)))
		if errors.Is(err, storage.ErrNoteChanged) {
			m.afterChange()
			m.c.setStatus("%s", i18n.T("La nota cambió por fuera: se recargó, vuelve a pegar", "The note changed outside: reloaded, paste again"))
			return
		}
		m.c.errStatus("No se pudo insertar la imagen", "Could not insert the image", err)
		return
	}
	m.afterChange()
	m.notes.selectPath(path)
	if task != nil {
		m.tasks.selectTask(path, task.Line)
	}
	m.c.setStatus(i18n.T("Imagen guardada en %s", "Image saved to %s"), ref)
}

// handlePaste procesa un pegado (bracketed paste). Con el popup de nombre
// abierto el texto va al campo; si no, solo se aceptan rutas de archivos de
// imagen (Cmd+V sobre un archivo copiado en el Finder), que se copian a assets/.
func (m *AppModel) handlePaste(msg tea.PasteMsg) tea.Cmd {
	if p := m.c.top(); p != nil {
		if ip, ok := p.(*inputPopup); ok {
			return ip.paste(msg)
		}
		return nil
	}
	paths := clipboard.ParsePastedPaths(msg.Content)
	if len(paths) == 0 {
		m.c.setStatus("%s", i18n.T("Lo pegado no es una imagen: usa Ctrl+V o copia el archivo de imagen", "Pasted text is not an image: use Ctrl+V or copy the image file"))
		return nil
	}
	for _, src := range paths {
		m.attachImage(func(dir, name string) (string, error) { return m.c.clip.ImportFile(dir, name, src) })
	}
	return nil
}
