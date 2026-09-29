package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Task representa un ítem de tarea de markdown (- [ ] o - [x])
type Task struct {
	NoteTitle string
	Line      int
	Text      string
	Done      bool
}

// Note representa un documento de Markdown
type Note struct {
	ID        string
	Title     string
	Path      string
	Content   string
	Tags      []string
	Category  string
	ModTime   time.Time
	Size      int64
	Tasks     []Task
	Images    []string
}

// Storage maneja el acceso y persistencia en el sistema de archivos
type Storage struct {
	BaseDir string
}

func New(baseDir string) *Storage {
	return &Storage{BaseDir: baseDir}
}

var (
	taskRegex     = regexp.MustCompile(`^[-*]\s+\[([ xX])\]\s+(.*)$`)
	imageRegex    = regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)
	tagRegex      = regexp.MustCompile(`#([a-zA-Z0-9_-]+)`)
	unsafeChars   = regexp.MustCompile(`[\\/:*?"<>|]`)
)

// ListNotes escanea el directorio y devuelve todas las notas .md ordenadas por fecha de modificación
func (s *Storage) ListNotes() ([]Note, error) {
	var notes []Note

	entries, err := os.ReadDir(s.BaseDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}

		fullPath := filepath.Join(s.BaseDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		contentBytes, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		content := string(contentBytes)

		title := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		title = strings.ReplaceAll(title, "-", " ")
		title = strings.ReplaceAll(title, "_", " ")

		note := Note{
			ID:      entry.Name(),
			Title:   title,
			Path:    fullPath,
			Content: content,
			ModTime: info.ModTime(),
			Size:    info.Size(),
		}

		// Extraer tags, tareas e imágenes
		note.Tags = s.extractTags(content)
		note.Tasks = s.extractTasks(note.Title, content)
		note.Images = s.extractImages(content)

		notes = append(notes, note)
	}

	// Ordenar por fecha de modificación descendente (la más reciente primero)
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

func (s *Storage) extractTasks(title, content string) []Task {
	var tasks []Task
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 1
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if matches := taskRegex.FindStringSubmatch(line); len(matches) == 3 {
			done := matches[1] == "x" || matches[1] == "X"
			tasks = append(tasks, Task{
				NoteTitle: title,
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

// CreateNote crea una nueva nota en blanco o con plantilla
func (s *Storage) CreateNote(title string) (*Note, error) {
	cleanName := strings.ToLower(title)
	cleanName = strings.ReplaceAll(cleanName, " ", "-")
	cleanName = unsafeChars.ReplaceAllString(cleanName, "")
	fileName := fmt.Sprintf("%s.md", cleanName)
	fullPath := filepath.Join(s.BaseDir, fileName)

	if _, err := os.Stat(fullPath); err == nil {
		return nil, fmt.Errorf("ya existe una nota con el nombre: %s", fileName)
	}

	initialContent := fmt.Sprintf("# %s\n\nFecha: %s\nTags: #general\n\n- [ ] Primera tarea pendiente\n",
		title, time.Now().Format("2006-01-02 15:04"))

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

// DeleteNote elimina una nota del disco
func (s *Storage) DeleteNote(path string) error {
	return os.Remove(path)
}
