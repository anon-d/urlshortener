// Package audit реализует подсистему аудита событий сервиса.
package audit

import "sync"

// AuditEvent описывает событие аудита.
type AuditEvent struct {
	Timestamp int64  `json:"ts"`
	Action    string `json:"action"`
	UserID    string `json:"user_id,omitempty"`
	URL       string `json:"url"`
}

// Observer — интерфейс наблюдателя, получающего события аудита.
type Observer interface {
	Notify(event AuditEvent)
}

// Publisher — издатель, рассылающий события всем зарегистрированным наблюдателям.
type Publisher struct {
	mu        sync.RWMutex
	observers []Observer
}

// NewPublisher создаёт новый Publisher.
func NewPublisher() *Publisher {
	return &Publisher{}
}

// Subscribe добавляет наблюдателя.
func (p *Publisher) Subscribe(o Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, o)
}

// Publish рассылает событие всем наблюдателям.
func (p *Publisher) Publish(event AuditEvent) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, o := range p.observers {
		o.Notify(event)
	}
}
