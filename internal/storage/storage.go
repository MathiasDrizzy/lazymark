package storage

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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
	Images   []string
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
type Storage struct {
	BaseDir       string
	CurrentSubDir string
}

func New(baseDir string) *Storage {
	return &Storage{BaseDir: baseDir, CurrentSubDir: ""}
}

var (
	taskRegex   = regexp.MustCompile(`^[-*]\s+\[([ xX])\]\s+(.*)$`)
	imageRegex  = regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)
	tagRegex    = regexp.MustCompile(`#([a-zA-Z0-9_-]+)`)
	unsafeChars = regexp.MustCompile(`[\\/:*?"<>|]`)
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
			info, err := de.Info()
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
				contentBytes, err := os.ReadFile(fullPath)
				if err != nil {
					continue
				}
				content := string(contentBytes)
				title := strings.TrimSuffix(name, filepath.Ext(name))
				title = strings.ReplaceAll(title, "-", " ")
				title = strings.ReplaceAll(title, "_", " ")

				note := Note{
					ID:      name,
					Title:   title,
					Path:    fullPath,
					Content: content,
					ModTime: info.ModTime(),
					Size:    info.Size(),
				}
				note.Tags = s.extractTags(content)
				note.Tasks = s.extractTasks(note.Title, fullPath, content)
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

		info, err := d.Info()
		if err != nil {
			return nil
		}

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(contentBytes)
		title := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		title = strings.ReplaceAll(title, "-", " ")
		title = strings.ReplaceAll(title, "_", " ")

		note := Note{
			ID:      d.Name(),
			Title:   title,
			Path:    path,
			Content: content,
			ModTime: info.ModTime(),
			Size:    info.Size(),
		}
		note.Tags = s.extractTags(content)
		note.Tasks = s.extractTasks(note.Title, path, content)
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
	matches := tagRegex.FindAllStringSubmatch(content, -1)
	tagMap := make(map[string]bool)
	for _, m := range matches {
		if len(m) > 1 {
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
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 1
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if matches := taskRegex.FindStringSubmatch(line); len(matches) == 3 {
			done := matches[1] == "x" || matches[1] == "X"
			tasks = append(tasks, Task{
				NoteTitle: title,
				NotePath:  path,
				Line:      lineNum,
				Text:      matches[2],
				Done:      done,
			})
		}
		lineNum++
	}
	return tasks
}

func (s *Storage) extractImages(content string) []string {
	matches := imageRegex.FindAllStringSubmatch(content, -1)
	var images []string
	for _, m := range matches {
		if len(m) > 2 {
			images = append(images, m[2])
		}
	}
	return images
}

// CreateNoteInDir crea una nueva nota en blanco en el directorio indicado
func (s *Storage) CreateNoteInDir(dir, title string) (*Note, error) {
	if dir == "" {
		dir = s.BaseDir
	}
	cleanName := strings.ToLower(title)
	cleanName = strings.ReplaceAll(cleanName, " ", "-")
	cleanName = unsafeChars.ReplaceAllString(cleanName, "")
	fileName := fmt.Sprintf("%s.md", cleanName)
	fullPath := filepath.Join(dir, fileName)

	if _, err := os.Stat(fullPath); err == nil {
		return nil, fmt.Errorf("ya existe una nota con el nombre: %s", fileName)
	}

	initialContent := fmt.Sprintf("# %s\n\n%s: %s\nTags: #general\n\n- [ ] %s\n",
		title, i18n.T("Fecha", "Date"), time.Now().Format("2006-01-02 15:04"), i18n.T("Primera tarea pendiente", "First pending task"))

	if err := os.WriteFile(fullPath, []byte(initialContent), 0644); err != nil {
		return nil, err
	}

	return &Note{
		ID:      fileName,
		Title:   title,
		Path:    fullPath,
		Content: initialContent,
		ModTime: time.Now(),
		Tags:    []string{"general"},
	}, nil
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
	cleanName := strings.ToLower(name)
	cleanName = strings.ReplaceAll(cleanName, " ", "-")
	cleanName = unsafeChars.ReplaceAllString(cleanName, "")
	fullPath := filepath.Join(parentDir, cleanName)
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

var toggleTaskRegex = regexp.MustCompile(`^(\s*[-*]\s+\[)([ xX])(\]\s*.*)$`)

// ToggleTask modifica de forma atómica el estado de una tarea (- [ ] <-> - [x]) en el archivo markdown
func (s *Storage) ToggleTask(notePath string, lineNum int) (bool, error) {
	contentBytes, err := os.ReadFile(notePath)
	if err != nil {
		return false, fmt.Errorf("error al leer archivo para alternar tarea: %w", err)
	}

	info, err := os.Stat(notePath)
	if err != nil {
		return false, fmt.Errorf("error al obtener info de archivo: %w", err)
	}

	lines := strings.Split(string(contentBytes), "\n")
	targetIdx := lineNum - 1
	if targetIdx < 0 || targetIdx >= len(lines) {
		return false, fmt.Errorf("índice de línea %d fuera de rango", lineNum)
	}

	matches := toggleTaskRegex.FindStringSubmatch(lines[targetIdx])
	if len(matches) != 4 {
		return false, fmt.Errorf("la línea %d no es una tarea válida de markdown", lineNum)
	}

	isDone := matches[2] == "x" || matches[2] == "X"
	newDone := !isDone

	newMark := " "
	if newDone {
		newMark = "x"
	}
	lines[targetIdx] = matches[1] + newMark + matches[3]

	newContent := strings.Join(lines, "\n")
	tmpPath := notePath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(newContent), info.Mode().Perm()); err != nil {
		return false, fmt.Errorf("error al escribir archivo temporal: %w", err)
	}

	if err := os.Rename(tmpPath, notePath); err != nil {
		_ = os.Remove(tmpPath)
		return false, fmt.Errorf("error al renombrar archivo atómico: %w", err)
	}

	return newDone, nil
}

// TaskStage define la columna del tablero Kanban (To Do, In Progress, Done)
type TaskStage int

const (
	StageTodo TaskStage = iota // 0: Por Hacer (- [ ])
	StageDoing                 // 1: En Progreso (- [ ] con #doing, #wip, #progreso)
	StageDone                  // 2: Completado (- [x])
)

var inProgressTagRegex = regexp.MustCompile(`(?i)#(doing|wip|progreso|in-progress)\b`)

// IsTaskDoing determina si una tarea está en progreso basándose en sus etiquetas
func IsTaskDoing(taskText string) bool {
	return inProgressTagRegex.MatchString(taskText)
}

// CleanTaskText devuelve el texto de la tarea sin las etiquetas de control Kanban (#doing, #wip)
func CleanTaskText(taskText string) string {
	cleaned := inProgressTagRegex.ReplaceAllString(taskText, "")
	return strings.TrimSpace(cleaned)
}

// GetTaskStage devuelve la etapa Kanban de una tarea
func GetTaskStage(task Task) TaskStage {
	if task.Done {
		return StageDone
	}
	if IsTaskDoing(task.Text) {
		return StageDoing
	}
	return StageTodo
}

// UpdateTaskStage actualiza de forma atómica en disco el estado Kanban de una tarea
func (s *Storage) UpdateTaskStage(notePath string, lineNum int, targetStage TaskStage) error {
	contentBytes, err := os.ReadFile(notePath)
	if err != nil {
		return fmt.Errorf("error al leer archivo para actualizar etapa: %w", err)
	}

	info, err := os.Stat(notePath)
	if err != nil {
		return fmt.Errorf("error al obtener info de archivo: %w", err)
	}

	lines := strings.Split(string(contentBytes), "\n")
	targetIdx := lineNum - 1
	if targetIdx < 0 || targetIdx >= len(lines) {
		return fmt.Errorf("índice de línea %d fuera de rango", lineNum)
	}

	matches := toggleTaskRegex.FindStringSubmatch(lines[targetIdx])
	if len(matches) != 4 {
		return fmt.Errorf("la línea %d no es una tarea válida de markdown", lineNum)
	}

	rest := strings.TrimPrefix(matches[3], "]")
	restTrimmed := strings.TrimSpace(rest)
	cleanText := CleanTaskText(restTrimmed)

	var newLine string
	switch targetStage {
	case StageTodo:
		newLine = fmt.Sprintf("%s ] %s", matches[1], cleanText)
	case StageDoing:
		newLine = fmt.Sprintf("%s ] %s #doing", matches[1], cleanText)
	case StageDone:
		newLine = fmt.Sprintf("%sx] %s", matches[1], cleanText)
	}

	lines[targetIdx] = newLine
	newContent := strings.Join(lines, "\n")
	tmpPath := notePath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(newContent), info.Mode().Perm()); err != nil {
		return fmt.Errorf("error al escribir archivo temporal: %w", err)
	}

	if err := os.Rename(tmpPath, notePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("error al renombrar archivo atómico: %w", err)
	}

	return nil
}
