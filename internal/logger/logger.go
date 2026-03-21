// Package logger предоставляет инициализацию структурированного логгера на основе zap.
package logger

import (
	"log"

	"go.uber.org/zap"
)

// New создаёт production-логгер zap.SugaredLogger.
func New() (*zap.SugaredLogger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Printf("Init logger: error %v", err)
		return nil, err
	}
	sugar := logger.Sugar()
	return sugar, nil
}
