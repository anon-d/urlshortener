package worker

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// DeleteRequest представляет запрос на удаление URL
type DeleteRequest struct {
	UserID   string
	ShortURL string
}

// DeleteStorage интерфейс для batch удаления
type DeleteStorage interface {
	BatchMarkAsDeleted(ctx context.Context, requests []DeleteRequest) error
}

// DeleteWorker обрабатывает запросы на удаление с буферизацией
type DeleteWorker struct {
	storage       DeleteStorage
	inputChannels []<-chan DeleteRequest
	bufferSize    int
	flushInterval time.Duration
	logger        *zap.SugaredLogger
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewDeleteWorker создает новый worker для удаления
func NewDeleteWorker(
	storage DeleteStorage,
	bufferSize int,
	flushInterval time.Duration,
	logger *zap.SugaredLogger,
) *DeleteWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &DeleteWorker{
		storage:       storage,
		inputChannels: make([]<-chan DeleteRequest, 0),
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// AddChannel добавляет новый канал для обработки
func (w *DeleteWorker) AddChannel(ch <-chan DeleteRequest) {
	w.inputChannels = append(w.inputChannels, ch)
}

func (w *DeleteWorker) Start() {
	mergedChan := w.fanIn(w.inputChannels...)

	go w.processDeletes(mergedChan)
}

// fanIn объединяет несколько каналов в один
func (w *DeleteWorker) fanIn(channels ...<-chan DeleteRequest) <-chan DeleteRequest {
	out := make(chan DeleteRequest, w.bufferSize)
	var wg sync.WaitGroup

	for _, ch := range channels {
		wg.Add(1)
		go func(c <-chan DeleteRequest) {
			defer wg.Done()
			for {
				select {
				case <-w.ctx.Done():
					return
				case req, ok := <-c:
					if !ok {
						return
					}
					select {
					case out <- req:
					case <-w.ctx.Done():
						return
					}
				}
			}
		}(ch)
	}

	// Закрытие выходного канала после завершения всех входных
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// processDeletes обрабатывает запросы с буферизацией
func (w *DeleteWorker) processDeletes(input <-chan DeleteRequest) {
	buffer := make([]DeleteRequest, 0, w.bufferSize)
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			if len(buffer) > 0 {
				w.flush(buffer)
			}
			return

		case req, ok := <-input:
			if !ok {
				if len(buffer) > 0 {
					w.flush(buffer)
				}
				return
			}

			buffer = append(buffer, req)

			// Если буфер заполнен, сразу отправляем в БД
			if len(buffer) >= w.bufferSize {
				w.flush(buffer)
				buffer = buffer[:0]
			}

		case <-ticker.C:
			// По таймеру отправляем накопленное
			if len(buffer) > 0 {
				w.flush(buffer)
				buffer = buffer[:0]
			}
		}
	}
}

// flush выполняет batch update в БД
func (w *DeleteWorker) flush(requests []DeleteRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := w.storage.BatchMarkAsDeleted(ctx, requests); err != nil {
		w.logger.Errorw("failed to batch delete URLs", "error", err, "count", len(requests))
		return
	}

	w.logger.Infow("batch deleted URLs", "count", len(requests))
}

func (w *DeleteWorker) Stop() {
	w.cancel()
}
