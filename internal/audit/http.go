package audit

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// HTTPObserver отправляет события аудита на удалённый сервер методом POST.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver создаёт HTTPObserver для указанного URL.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Notify отправляет событие аудита POST-запросом.
func (h *HTTPObserver) Notify(event AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("audit http: failed to marshal %q event: %v", event.Action, err)
		return
	}

	resp, err := h.client.Post(h.url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("audit http: failed to send %q event to %s: %v", event.Action, h.url, err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
}
