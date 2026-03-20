package audit

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

// FileObserver записывает события аудита в файл (append).
type FileObserver struct {
	mu   sync.Mutex
	path string
}

// NewFileObserver создаёт FileObserver для указанного пути файла.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

// Notify записывает событие в конец файла на новой строке.
func (f *FileObserver) Notify(event AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("audit file: failed to marshal event: %v", err)
		return
	}
	data = append(data, '\n')

	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("audit file: failed to open file %s: %v", f.path, err)
		return
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		log.Printf("audit file: failed to write event: %v", err)
	}
}
