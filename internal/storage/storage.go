package storage

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/safeio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// Task representa un ítem de tarea de markdown (- [ ] o - [x])
type Task struct {
	NoteTitle string
	NotePath  string
	Line      int
	Text      string
	Done      bool
	Indent    int   // ancho de la sangría de su línea (tab hasta el múltiplo de 4); 0 en el margen
	Dates     Dates // inicio, vencimiento y completada (🛫 📅 ✅), leídas del texto
}

// Note representa un documento de Markdown
type Note struct {
	ID       string
	Title    string
	Path     string
	Content  string
	Tags     []string
	Category string
	ModTime  time.Time
	Size     int64
	Tasks    []Task
	TooLarge bool // pesa más de MaxNoteBytes: no se leyó (Content vacío)
	Images   []string
	// Warnings son avisos de la creación de la nota (hoy: las {{variables}} desconocidas de su plantilla); no se guardan.
	Warnings []string
}

// EntryType define si es una nota o una carpeta
type EntryType int

const (
	EntryNote EntryType = iota
	EntryFolder
)

// NoteEntry representa una fila en el explorador de notas en árbol (carpeta o archivo)
type NoteEntry struct {
	Type     EntryType
	Name     string
	Path     string
	Note     *Note
	ModTime  time.Time
	Depth    int  // Nivel de anidamiento en el árbol (0 para raíz, 1 para hijos, etc.)
	Expanded bool // Si es carpeta, indica si sus hijos están visibles
	Children int  // Cantidad de elementos dentro de la carpeta
}

// Storage maneja el acceso y persistencia en el sistema de archivos
// MaxNoteBytes es el tamaño máximo de una nota que se lee (config max_note_mb, 10 MB por defecto). Una nota más grande se lista sin su contenido
// (Note.TooLarge) y las operaciones que la leerían entera dan ErrNoteTooLarge: una nota de cientos de MB no se carga en memoria.
var MaxNoteBytes int64 = 10 << 20

// ErrNoteTooLarge es el error de leer una nota que pesa más de MaxNoteBytes.
var ErrNoteTooLarge = safeio.ErrTooLarge

type Storage struct {
	BaseDir       string
	CurrentSubDir string
	// DateFormatPref es la configuración date_format: "dataview" o "emoji" fijan el formato en que se escriben las fechas; "" (por defecto) es dataview,
	// salvo en un vault que ya tiene tareas con emojis y ninguna con Dataview (ahí se escribe en emoji para no mezclar formatos y se avisa una vez).
	DateFormatPref string
	// DateNoticeSeen indica que el aviso del formato ya se mostró alguna vez (se guarda en la configuración).
	DateNoticeSeen bool

	fmtMu     sync.Mutex
	fmtKnown  bool
	fmtVault  DateFormat
	fmtExcept bool   // el formato salió de la excepción del vault (solo emojis)
	notice    string // aviso pendiente de mostrar (TakeDateFormatNotice)

	trashIssues []string // entradas de trash.json que se ignoraron por inválidas (TrashIssues)
}

func New(baseDir string) *Storage {
	return &Storage{BaseDir: baseDir, CurrentSubDir: ""}
}

var wikilinkRe = regexp.MustCompile(`\[\[[^\[\]\n]*\]\]`)

var (
	// una tarea: viñeta (- * +) o número de lista (1. 1)) y casilla; la sangría ya se quitó
	taskRegex     = regexp.MustCompile(`^(?:[-*+]|\d+[.)])\s+\[([ xX])\]\s+(.*)$`)
	listItemRegex = regexp.MustCompile(`^(?:[-*+]|\d+[.)])(?:\s|$)`)
	imageRegex    = regexp.MustCompile(`!\[(.*?)\]\(((?:\\.|[^\\)])*?)\)`)
	tagRegex      = regexp.MustCompile(`#([a-zA-Z0-9_-]+)(/[a-zA-Z0-9_/-]*)?`)
	unsafeChars   = regexp.MustCompile(`[\\/:*?"<>|\[\]#^]`) // además de lo que el sistema de archivos no admite, lo que rompe un [[wikilink]]
)

// CurrentDir devuelve la ruta absoluta del directorio actualmente navegado
func (s *Storage) CurrentDir() string {
	if s.CurrentSubDir == "" {
		return s.BaseDir
	}
	return filepath.Join(s.BaseDir, s.CurrentSubDir)
}

// CountFolderItems cuenta cuántos elementos directos (carpetas o archivos .md) contiene una carpeta
func (s *Storage) CountFolderItems(folderPath string) int {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || strings.EqualFold(name, "assets") {
			continue
		}
		if e.IsDir() || strings.HasSuffix(strings.ToLower(name), ".md") {
			count++
		}
	}
	return count
}

// ListTreeEntries devuelve la estructura jerárquica de carpetas y notas en árbol, respetando carpetas expandidas
func (s *Storage) ListTreeEntries(expanded map[string]bool) ([]NoteEntry, error) {
	var result []NoteEntry

	var walk func(dirPath string, depth int) error
	walk = func(dirPath string, depth int) error {
		dirEntries, err := os.ReadDir(dirPath)
		if err != nil {
			return err
		}

		var folderEntries []NoteEntry
		var noteEntries []NoteEntry

		for _, de := range dirEntries {
			name := de.Name()
			if strings.HasPrefix(name, ".") || strings.EqualFold(name, "assets") {
				continue
			}

			fullPath := filepath.Join(dirPath, name)
			info, err := s.entryInfo(fullPath, de)
			if err != nil {
				continue
			}

			if de.IsDir() {
				childrenCount := s.CountFolderItems(fullPath)
				isExpanded := true
				if expanded != nil {
					if val, ok := expanded[fullPath]; ok {
						isExpanded = val
					}
				}

				folderEntries = append(folderEntries, NoteEntry{
					Type:     EntryFolder,
					Name:     name,
					Path:     fullPath,
					ModTime:  info.ModTime(),
					Depth:    depth,
					Expanded: isExpanded,
					Children: childrenCount,
				})
			} else if strings.HasSuffix(strings.ToLower(name), ".md") {
				if !s.linkStaysInside(fullPath, de) {
					continue
				}
				contentBytes, err := safeio.ReadRegular(fullPath, MaxNoteBytes)
				tooLarge := errors.Is(err, safeio.ErrTooLarge)
				if err != nil && !tooLarge { // un FIFO, un dispositivo o una nota ilegible no se listan
					continue
				}
				content := string(contentBytes)
				title := strings.TrimSuffix(name, filepath.Ext(name))
				title = strings.ReplaceAll(title, "-", " ")
				title = strings.ReplaceAll(title, "_", " ")

				note := Note{
					ID:       name,
					Title:    title,
					Path:     fullPath,
					Content:  content,
					ModTime:  info.ModTime(),
					Size:     info.Size(),
					TooLarge: tooLarge,
				}
				if !s.inTemplates(fullPath) { // una plantilla es un molde: sus etiquetas y casillas no cuentan
					note.Tags = s.extractTags(content)
					note.Tasks = s.extractTasks(note.Title, fullPath, content)
				}
				note.Images = s.extractImages(content)

				noteEntries = append(noteEntries, NoteEntry{
					Type:    EntryNote,
					Name:    name,
					Path:    fullPath,
					Note:    &note,
					ModTime: info.ModTime(),
					Depth:   depth,
				})
			}
		}

		sort.Slice(folderEntries, func(i, j int) bool {
			return strings.ToLower(folderEntries[i].Name) < strings.ToLower(folderEntries[j].Name)
		})

		sort.Slice(noteEntries, func(i, j int) bool {
			return strings.ToLower(noteEntries[i].Name) < strings.ToLower(noteEntries[j].Name)
		})

		for _, f := range folderEntries {
			result = append(result, f)
			if f.Expanded {
				_ = walk(f.Path, depth+1)
			}
		}

		result = append(result, noteEntries...)
		return nil
	}

	err := walk(s.BaseDir, 0)
	return result, err
}

// ListEntries lista carpetas y notas en árbol para el explorador
func (s *Storage) ListEntries() ([]NoteEntry, error) {
	return s.ListTreeEntries(nil)
}

// ListNotes escanea recursivamente todas las notas .md del BaseDir para tags y tareas globales
func (s *Storage) ListNotes() ([]Note, error) {
	var notes []Note

	err := filepath.WalkDir(s.BaseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if (strings.HasPrefix(name, ".") && name != ".") || strings.EqualFold(name, "assets") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		if !s.linkStaysInside(path, d) {
			return nil
		}

		info, err := s.entryInfo(path, d)
		if err != nil {
			return nil
		}

		contentBytes, err := safeio.ReadRegular(path, MaxNoteBytes)
		tooLarge := errors.Is(err, safeio.ErrTooLarge)
		if err != nil && !tooLarge { // un FIFO, un dispositivo o una nota ilegible no se listan
			return nil
		}
		content := string(contentBytes)
		title := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		title = strings.ReplaceAll(title, "-", " ")
		title = strings.ReplaceAll(title, "_", " ")

		note := Note{
			ID:       d.Name(),
			Title:    title,
			Path:     path,
			Content:  content,
			ModTime:  info.ModTime(),
			Size:     info.Size(),
			TooLarge: tooLarge,
		}
		if !s.inTemplates(path) { // una plantilla es un molde, no una nota: sus etiquetas y casillas no cuentan
			note.Tags = s.extractTags(content)
			note.Tasks = s.extractTasks(note.Title, path, content)
		}
		note.Images = s.extractImages(content)

		notes = append(notes, note)
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].ModTime.After(notes[j].ModTime)
	})

	return notes, nil
}

func (s *Storage) extractTags(content string) []string {
	matches := tagRegex.FindAllStringSubmatch(wikilinkRe.ReplaceAllString(content, ""), -1) // el # de [[nota#Título]] no es una etiqueta
	tagMap := make(map[string]bool)
	for _, m := range matches {
		if len(m) > 1 {
			if strings.EqualFold(m[1], "kb") && len(m) > 2 && m[2] != "" {
				continue // #kb/<columna> es del tablero, no una categoría
			}
			tagMap[strings.ToLower(m[1])] = true
		}
	}
	var tags []string
	for t := range tagMap {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

func (s *Storage) extractTasks(title, path, content string) []Task {
	var tasks []Task
	var (
		fence       string // el vallado de código abierto (``` o ~~~, con su largo), o ""
		fenceIndent int    // la sangría con que se abrió: una línea con menos cierra el ítem que lo contenía, y con él el vallado
		fenceInList bool   // se abrió dentro de una lista: ahí un vallado puede llevar 4 o más espacios de sangría
		inList      bool   // estamos dentro de una lista: ahí una sangría de 4 o más es una sublista y no un bloque de código
	)
	for i, raw := range strings.Split(content, "\n") {
		line := strings.TrimSuffix(raw, "\r")
		body := strings.TrimLeft(line, " \t")
		indent := indentWidth(line[:len(line)-len(body)])
		if fence != "" && body != "" && indent < fenceIndent {
			fence = "" // CommonMark: un vallado sin cerrar termina con el ítem de lista que lo contiene
		}
		if f := fenceMarker(body); f != "" && (indent < 4 || inList || fence != "") {
			switch {
			case fence == "":
				fence, fenceIndent, fenceInList = f, indent, inList
				if indent == 0 {
					inList = false // un vallado al margen corta la lista
				}
			case f[0] == fence[0] && len(f) >= len(fence) && strings.TrimSpace(body)[len(f):] == "" && (indent < 4 || fenceInList):
				fence = ""
			}
			continue
		}
		if fence != "" { // lo que está dentro de un bloque de código no es una tarea
			continue
		}
		if body == "" { // las líneas en blanco no cortan una lista
			continue
		}
		item := listItemRegex.MatchString(body)
		switch {
		case item && indent < 4:
			inList = true
		case !item && indent == 0:
			inList = false // un párrafo pegado al margen termina la lista
		}
		if !item || (indent >= 4 && !inList) { // sangría de 4 fuera de una lista: es un bloque de código, no una sublista
			continue
		}
		if m := taskRegex.FindStringSubmatch(body); len(m) == 3 {
			tasks = append(tasks, Task{
				NoteTitle: title,
				NotePath:  path,
				Line:      i + 1,
				Text:      m[2],
				Done:      m[1] == "x" || m[1] == "X",
				Indent:    indent,
				Dates:     ParseDates(m[2]),
			})
		}
	}
	return tasks
}

// indentWidth es el ancho de una sangría: un espacio es 1 y un tabulador llega al siguiente múltiplo de 4.
func indentWidth(ws string) int {
	w := 0
	for _, r := range ws {
		if r == '\t' {
			w += 4 - w%4
		} else {
			w++
		}
	}
	return w
}

// FenceMarker devuelve el vallado de código con el que empieza la línea (ver fenceMarker), o "".
func FenceMarker(line string) string { return fenceMarker(line) }

// fenceMarker devuelve el vallado de código (3 o más ` o ~) con el que empieza la línea, o "".
func fenceMarker(line string) string {
	t := strings.TrimLeft(line, " \t")
	if len(t) < 3 || (t[0] != '`' && t[0] != '~') {
		return ""
	}
	i := 0
	for i < len(t) && t[i] == t[0] {
		i++
	}
	if i < 3 || (t[0] == '`' && strings.Contains(t[i:], "`")) {
		return ""
	}
	return t[:i]
}

func (s *Storage) extractImages(content string) []string {
	matches := imageRegex.FindAllStringSubmatch(content, -1)
	var images []string
	for _, m := range matches {
		if len(m) > 2 {
			images = append(images, UnescapeRef(m[2]))
		}
	}
	return images
}

// CreateNoteInDir crea una nueva nota en blanco en el directorio indicado
func (s *Storage) CreateNoteInDir(dir, title string) (*Note, error) {
	return s.CreateNoteInDirWithBody(dir, title, "")
}

// ErrNoteExists es el error de crear una nota o carpeta cuyo nombre ya está ocupado (por un archivo, una carpeta o
// un enlace simbólico, también colgante). No se escribió nada.
var ErrNoteExists = i18n.NewError("ya existe", "already exists")

// CreateNoteInDirWithBody crea la nota con body como contenido (vacío: la plantilla con fecha y primera tarea). Se
// crea con O_EXCL: no sigue enlaces simbólicos ni pisa nada, y si el nombre está ocupado no escribe.
func (s *Storage) CreateNoteInDirWithBody(dir, title, body string) (*Note, error) {
	if dir == "" {
		dir = s.BaseDir
	}
	cleanName := slug(title)
	if cleanName == "" {
		return nil, i18n.Errorf("nombre de nota vacío", "empty note name")
	}
	fileName := fmt.Sprintf("%s.md", cleanName)
	fullPath := filepath.Join(dir, fileName)

	content := body
	if content == "" {
		content = fmt.Sprintf("# %s\n\n%s: %s\nTags: #general\n\n- [ ] %s\n",
			title, i18n.T("Fecha", "Date"), time.Now().Format("2006-01-02 15:04"), i18n.T("Primera tarea pendiente", "First pending task"))
	}

	f, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, i18n.Errorf("%w una nota con el nombre: %s", "%w: a note named %s", ErrNoteExists, fileName)
		}
		return nil, err
	}
	_, err = f.WriteString(content)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, err
	}

	note := &Note{ID: fileName, Title: title, Path: fullPath, Content: content, ModTime: time.Now()}
	note.Tags = s.extractTags(content)
	note.Tasks = s.extractTasks(title, fullPath, content)
	return note, nil
}

// CreateNote crea una nueva nota en blanco o con plantilla en el directorio actual o raíz
func (s *Storage) CreateNote(title string) (*Note, error) {
	return s.CreateNoteInDir(s.CurrentDir(), title)
}

// CreateFolderInDir crea una subcarpeta dentro del directorio padre indicado
func (s *Storage) CreateFolderInDir(parentDir, name string) (string, error) {
	if parentDir == "" {
		parentDir = s.BaseDir
	}
	cleanName := slug(name)
	if cleanName == "" {
		return "", i18n.Errorf("nombre de carpeta vacío", "empty folder name")
	}
	fullPath := filepath.Join(parentDir, cleanName)
	if _, err := os.Lstat(fullPath); err == nil { // Lstat: un enlace simbólico (también colgante) cuenta como ocupado
		return "", fmt.Errorf("%w: %s", ErrNoteExists, cleanName)
	}
	return fullPath, os.MkdirAll(fullPath, 0755)
}

// CreateFolder crea una subcarpeta dentro del directorio actual o raíz
func (s *Storage) CreateFolder(name string) error {
	_, err := s.CreateFolderInDir(s.CurrentDir(), name)
	return err
}

// DeleteNote elimina una nota o carpeta del disco
func (s *Storage) DeleteNote(path string) error {
	return os.RemoveAll(path)
}

// MoveNote traslada una nota a la carpeta de destino especificada
func (s *Storage) MoveNote(notePath, targetFolderPath string) error {
	baseName := filepath.Base(notePath)
	destPath := filepath.Join(targetFolderPath, baseName)
	if destPath == notePath {
		return nil
	}
	if _, err := os.Stat(destPath); err == nil {
		return i18n.Errorf("ya existe %s en el destino", "%s already exists at destination", baseName)
	}
	return os.Rename(notePath, destPath)
}

// ListFolders lista todas las carpetas disponibles en BaseDir (incluyendo la raíz)
func (s *Storage) ListFolders() ([]string, error) {
	var folders []string
	folders = append(folders, s.BaseDir)

	err := filepath.Walk(s.BaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || strings.EqualFold(name, "assets") {
				return filepath.SkipDir
			}
			if path != s.BaseDir {
				folders = append(folders, path)
			}
		}
		return nil
	})
	return folders, err
}

var toggleTaskRegex = regexp.MustCompile(`^(\s*(?:[-*+]|\d+[.)])\s+\[)([ xX])(\]\s*.*)$`)

// ToggleTask alterna una tarea (- [ ] <-> - [x]) reescribiendo solo esa línea.
func (s *Storage) ToggleTask(notePath string, lineNum int) (bool, error) {
	return s.ToggleTaskIfUnchanged(notePath, lineNum, time.Time{})
}

// ToggleTaskIfUnchanged alterna la tarea solo si la nota conserva el mtime con
// el que se cargó (expected); si cambió por fuera devuelve ErrNoteChanged y no
// escribe nada (X10). Un expected cero omite esa comprobación.
func (s *Storage) ToggleTaskIfUnchanged(notePath string, lineNum int, expected time.Time) (bool, error) {
	notePath, err := s.ResolveNote(notePath)
	if err != nil {
		return false, err
	}
	var newDone bool
	err = rewriteLine(notePath, lineNum, expected, func(line string) (string, error) {
		m := toggleTaskRegex.FindStringSubmatch(line)
		if len(m) != 4 {
			return "", i18n.Errorf("la línea %d no es una tarea válida de markdown", "line %d is not a valid markdown task", lineNum)
		}
		newDone = m[2] != "x" && m[2] != "X"
		mark := " "
		if newDone {
			mark = "x"
		}
		return withCompletionIn(m[1]+mark+m[3], newDone, s.WriteDateFormat()), nil
	})
	return newDone, err
}

// rewriteLine reemplaza la línea lineNum (desde 1) de la nota con fn(línea),
// dejando el resto del archivo idéntico byte a byte, y lo guarda de forma
// atómica. Si expected no es cero y el mtime en disco es otro, no escribe. Si
// la nota cambia entre la lectura y la escritura, tampoco (X10).
func rewriteLine(notePath string, lineNum int, expected time.Time, fn func(line string) (string, error)) error {
	return rewriteLines(notePath, expected, func(lines []string) ([]string, error) {
		idx := lineNum - 1
		if idx < 0 || idx >= len(lines) {
			return nil, i18n.Errorf("índice de línea %d fuera de rango", "line index %d out of range", lineNum)
		}
		newLine, err := fn(lines[idx])
		if err != nil {
			return nil, err
		}
		lines[idx] = newLine
		return lines, nil
	})
}

// rewriteLines es rewriteLine para una edición de varias líneas: edit recibe las líneas de la nota y devuelve las nuevas; el
// archivo se guarda de forma atómica y con las mismas comprobaciones del mtime.
func rewriteLines(notePath string, expected time.Time, edit func(lines []string) ([]string, error)) error {
	before, err := os.Stat(notePath)
	if err != nil {
		return i18n.Errorf("error al obtener info de archivo: %w", "could not read the file info: %w", err)
	}
	if !expected.IsZero() && !before.ModTime().Equal(expected) {
		return ErrNoteChanged
	}
	data, err := safeio.ReadRegular(notePath, MaxNoteBytes)
	if err != nil {
		return i18n.Errorf("error al leer la nota: %w", "could not read the note: %w", err)
	}
	lines, err := edit(strings.Split(string(data), "\n"))
	if err != nil {
		return err
	}

	// el temporal lleva un nombre aleatorio y se crea con O_EXCL en la misma carpeta: un enlace simbólico
	// preparado de antemano (`<nota>.md.tmp`) no puede desviar la escritura a otro archivo
	tmpFile, err := os.CreateTemp(filepath.Dir(notePath), ".lazymark-*.tmp")
	if err != nil {
		return i18n.Errorf("error al crear archivo temporal: %w", "could not create the temporary file: %w", err)
	}
	tmp := tmpFile.Name()
	_, werr := tmpFile.Write([]byte(strings.Join(lines, "\n")))
	if werr == nil {
		werr = safeio.SyncFile(tmpFile) // a disco antes del rename (A1)
	}
	cerr := tmpFile.Close()
	if werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Chmod(tmp, before.Mode().Perm())
	}
	if werr != nil {
		_ = os.Remove(tmp)
		return i18n.Errorf("error al escribir archivo temporal: %w", "could not write the temporary file: %w", werr)
	}
	after, err := os.Stat(notePath)
	if err != nil || !after.ModTime().Equal(before.ModTime()) || after.Size() != before.Size() {
		_ = os.Remove(tmp)
		return ErrNoteChanged
	}
	if err := os.Rename(tmp, notePath); err != nil {
		_ = os.Remove(tmp)
		return i18n.Errorf("error al renombrar archivo atómico: %w", "could not rename the temporary file: %w", err)
	}
	return nil
}

// Slug es el nombre de archivo (sin .md) que resulta de un nombre visible: el mismo que usan crear y renombrar.
func Slug(name string) string { return slug(name) }

// slug convierte un nombre visible en un nombre de archivo seguro.
func slug(name string) string {
	clean := strings.ToLower(strings.TrimSpace(name))
	clean = strings.TrimSuffix(clean, ".md")
	clean = unsafeChars.ReplaceAllString(clean, "")
	clean = strings.Join(strings.Fields(clean), "-")
	return strings.Trim(clean, ".-")
}

// Rename cambia el nombre de una nota o carpeta sin tocar su contenido y sin
// pisar otro archivo. Devuelve la ruta nueva.
func (s *Storage) Rename(path, newName string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	clean := slug(newName)
	if clean == "" {
		return "", i18n.Errorf("nombre vacío", "empty name")
	}
	if !fi.IsDir() {
		clean += ".md"
	}
	dest := filepath.Join(filepath.Dir(path), clean)
	if dest == path {
		return path, nil
	}
	if _, err := os.Lstat(dest); err == nil {
		return "", fmt.Errorf("%w: %s", ErrNoteExists, clean)
	}
	return dest, os.Rename(path, dest)
}

// ErrNoteChanged indica que la nota cambió en disco desde que se cargó.
var ErrNoteChanged = i18n.NewError("la nota cambió por fuera; recarga antes de editarla", "the note changed outside; reload before editing it")

// AppendToNote agrega text como un párrafo al final de la nota, sin tocar lo
// anterior. Si la nota cambió por fuera desde que se cargó (expected), no
// escribe y devuelve ErrNoteChanged (X10).
func (s *Storage) AppendToNote(notePath, text string, expected time.Time) error {
	notePath, err := s.ResolveNote(notePath)
	if err != nil {
		return err
	}
	fi, err := os.Stat(notePath)
	if err != nil {
		return err
	}
	if !expected.IsZero() && !fi.ModTime().Equal(expected) {
		return ErrNoteChanged
	}
	data, err := safeio.ReadRegular(notePath, MaxNoteBytes)
	if err != nil {
		return err
	}
	nl := "\n"
	if bytes.Contains(data, []byte("\r\n")) {
		nl = "\r\n"
	}
	var add string
	switch {
	case len(data) == 0:
		add = text + nl
	case bytes.HasSuffix(data, []byte("\n")):
		add = nl + text + nl
	default:
		add = nl + nl + text + nl
	}
	// atómico: el archivo nuevo (lo leído más lo agregado) se escribe aparte y se renombra encima, y solo si la nota sigue como se leyó
	return safeio.WriteFileAtomicIf(notePath, append(data, add...), 0o644, func() error {
		after, err := os.Stat(notePath) // justo antes del renombrado: si otro programa guardó mientras tanto, no se pisa
		if err != nil || !after.ModTime().Equal(fi.ModTime()) || after.Size() != fi.Size() {
			return ErrNoteChanged
		}
		return nil
	})
}

// InsertAfterLine inserta text, separado por una línea en blanco, debajo de la
// línea lineNum (desde 1), sin tocar el resto del archivo. Comprueba el mtime
// igual que AppendToNote.
func (s *Storage) InsertAfterLine(notePath string, lineNum int, text string, expected time.Time) error {
	notePath, err := s.ResolveNote(notePath)
	if err != nil {
		return err
	}
	return rewriteLine(notePath, lineNum, expected, func(line string) (string, error) {
		nl := "\n"
		if strings.HasSuffix(line, "\r") {
			nl = "\r\n"
			line = strings.TrimSuffix(line, "\r")
		}
		return line + nl + nl + text + strings.TrimSuffix(nl, "\n"), nil
	})
}

// ReplaceLineIf reemplaza la línea line (desde 1) de la nota por after solo si hoy es exactamente before (y, si expected no es cero, la
// nota conserva su mtime): así una edición calculada con una lectura vieja nunca pisa un cambio hecho por fuera. Escribe solo esa línea.
func (s *Storage) ReplaceLineIf(notePath string, line int, before, after string, expected time.Time) error {
	notePath, err := s.ResolveNote(notePath)
	if err != nil {
		return err
	}
	return rewriteLine(notePath, line, expected, func(cur string) (string, error) {
		if cur != before {
			return "", ErrNoteChanged
		}
		return after, nil
	})
}

// entryInfo es d.Info() salvo para un enlace simbólico: ahí se mira el destino (Stat), porque el mtime con el que se compara al escribir es el del archivo que se
// escribe, no el del enlace (si no, toda edición de una nota que es un enlace daba un falso "la nota cambió por fuera").
func (s *Storage) entryInfo(path string, d fs.DirEntry) (fs.FileInfo, error) {
	if d.Type()&fs.ModeSymlink != 0 {
		if fi, err := os.Stat(path); err == nil {
			return fi, nil
		}
	}
	return d.Info()
}

// refUnescaper deshace el escape de la referencia de una imagen: `\(`, `\)`, `\[`, `\]` y `%20` (lo que escribe `lazymark paste`).
var refUnescaper = strings.NewReplacer(`\(`, "(", `\)`, ")", `\[`, "[", `\]`, "]", "%20", " ")

// UnescapeRef devuelve la ruta de una imagen de markdown (`![](ruta)`) sin el escape de ( ) [ ] y espacios.
func UnescapeRef(ref string) string { return refUnescaper.Replace(ref) }
