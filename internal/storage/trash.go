package storage

import (
	"encoding/json"
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/safeio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const TrashRetentionDays = 20

// TrashItem representa un archivo o carpeta movido a la papelera
type TrashItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	OriginalPath string    `json:"original_path"`
	DeletedAt    time.Time `json:"deleted_at"`
	IsDir        bool      `json:"is_dir"`
	Size         int64     `json:"size"`
}

// DaysRemaining calcula cuántos días quedan antes de la eliminación permanente automática (20 días max)
func (t *TrashItem) DaysRemaining() int {
	elapsed := time.Since(t.DeletedAt)
	daysLeft := TrashRetentionDays - int(elapsed.Hours()/24)
	if daysLeft < 0 {
		return 0
	}
	return daysLeft
}

func (s *Storage) trashDir() string {
	return filepath.Join(s.BaseDir, ".trash")
}

func (s *Storage) trashMetaFile() string {
	return filepath.Join(s.trashDir(), "trash.json")
}

func (s *Storage) readTrashMeta() ([]TrashItem, error) {
	metaPath := s.trashMetaFile()
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []TrashItem{}, nil
		}
		return nil, err
	}
	var items []TrashItem
	if err := json.Unmarshal(data, &items); err != nil {
		return []TrashItem{}, nil
	}
	// trash.json vive dentro de la carpeta de notas: es contenido no confiable. Una entrada cuyo id no sea el
	// nombre de un archivo dentro de .trash/ se ignora y se reporta (TrashIssues); nunca se usa para borrar.
	valid := items[:0]
	s.trashIssues = nil
	for _, it := range items {
		if !validTrashID(it.ID) {
			s.trashIssues = append(s.trashIssues, fmt.Sprintf("papelera: id inválido %q (ignorado)", it.ID))
			continue
		}
		valid = append(valid, it)
	}
	return valid, nil
}

// validTrashID indica si id es un único nombre de archivo (sin separadores ni "..") y no el de los metadatos.
func validTrashID(id string) bool {
	if id == "" || id == "." || id == ".." || id == "trash.json" {
		return false
	}
	return !strings.ContainsAny(id, `/\`+"\x00") && id == filepath.Base(id) && filepath.VolumeName(id) == ""
}

// TrashIssues devuelve las entradas de la papelera que la última lectura ignoró por inválidas.
func (s *Storage) TrashIssues() []string { return append([]string(nil), s.trashIssues...) }

func (s *Storage) saveTrashMeta(items []TrashItem) error {
	td := s.trashDir()
	if err := os.MkdirAll(td, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return safeio.WriteFileAtomic(s.trashMetaFile(), data, 0o644)
}

// ListTrash devuelve los elementos de la papelera purgando automáticamente los que superen los 20 días
func (s *Storage) ListTrash() ([]TrashItem, error) {
	items, err := s.readTrashMeta()
	if err != nil {
		return nil, err
	}

	var validItems []TrashItem
	now := time.Now()
	cutoff := now.Add(-time.Duration(TrashRetentionDays) * 24 * time.Hour)
	hasChanges := false

	for _, item := range items {
		// Purgar elementos con más de 20 días
		if item.DeletedAt.Before(cutoff) {
			itemStoragePath := filepath.Join(s.trashDir(), item.ID)
			_ = os.RemoveAll(itemStoragePath)
			hasChanges = true
			continue
		}
		validItems = append(validItems, item)
	}

	if hasChanges {
		_ = s.saveTrashMeta(validItems)
	}

	// Ordenar de más reciente a más antiguo
	sort.Slice(validItems, func(i, j int) bool {
		return validItems[i].DeletedAt.After(validItems[j].DeletedAt)
	})

	return validItems, nil
}

// CountTrash devuelve la cantidad de elementos activos en la papelera
func (s *Storage) CountTrash() int {
	items, err := s.ListTrash()
	if err != nil {
		return 0
	}
	return len(items)
}

// MoveToTrash traslada una nota o carpeta a la papelera (.trash/) registrando su fecha de eliminación
func (s *Storage) MoveToTrash(targetPath string) (*TrashItem, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return nil, err
	}

	td := s.trashDir()
	if err := os.MkdirAll(td, 0755); err != nil {
		return nil, err
	}

	name := filepath.Base(targetPath)
	uniqueID := fmt.Sprintf("%d_%s", time.Now().UnixNano(), name)
	destPath := filepath.Join(td, uniqueID)

	if err := os.Rename(targetPath, destPath); err != nil {
		return nil, err
	}

	item := TrashItem{
		ID:           uniqueID,
		Name:         name,
		OriginalPath: targetPath,
		DeletedAt:    time.Now(),
		IsDir:        info.IsDir(),
		Size:         info.Size(),
	}

	items, _ := s.readTrashMeta()
	items = append(items, item)
	_ = s.saveTrashMeta(items)

	return &item, nil
}

// RestoreTrashItem restaura un elemento de la papelera a su ubicación original
func (s *Storage) RestoreTrashItem(id string) error {
	items, err := s.readTrashMeta()
	if err != nil {
		return err
	}

	var target *TrashItem
	var remaining []TrashItem
	for _, item := range items {
		if item.ID == id {
			target = &item
		} else {
			remaining = append(remaining, item)
		}
	}

	if target == nil {
		return fmt.Errorf("elemento no encontrado en la papelera")
	}

	srcPath := filepath.Join(s.trashDir(), target.ID)
	if err := s.confineNewPath(target.OriginalPath); err != nil { // original_path también viene del archivo
		return err
	}
	// Asegurar que el directorio de destino exista
	parentDir := filepath.Dir(target.OriginalPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return err
	}

	destPath := target.OriginalPath
	// Si ya existe un archivo con ese nombre, agregar sufijo restored
	if _, err := os.Stat(destPath); err == nil {
		ext := filepath.Ext(destPath)
		base := strings.TrimSuffix(destPath, ext)
		destPath = fmt.Sprintf("%s_restored_%d%s", base, time.Now().Unix(), ext)
	}

	if err := os.Rename(srcPath, destPath); err != nil {
		return err
	}

	return s.saveTrashMeta(remaining)
}

// DeleteTrashItem elimina permanentemente un elemento específico de la papelera
func (s *Storage) DeleteTrashItem(id string) error {
	if !validTrashID(id) {
		return fmt.Errorf("id de papelera inválido: %q", id)
	}
	items, err := s.readTrashMeta()
	if err != nil {
		return err
	}

	var remaining []TrashItem
	for _, item := range items {
		if item.ID != id {
			remaining = append(remaining, item)
		}
	}

	itemPath := filepath.Join(s.trashDir(), id)
	_ = os.RemoveAll(itemPath)

	return s.saveTrashMeta(remaining)
}

// EmptyTrash elimina definitivamente todos los elementos contenidos en la papelera
func (s *Storage) EmptyTrash() error {
	items, _ := s.readTrashMeta()
	for _, item := range items {
		itemPath := filepath.Join(s.trashDir(), item.ID)
		_ = os.RemoveAll(itemPath)
	}
	return s.saveTrashMeta([]TrashItem{})
}
