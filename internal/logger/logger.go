package logger

import (
	"log"

	"go.uber.org/zap"
)

func New() (*zap.SugaredLogger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Printf("Init logger: error %v", err)
		return nil, err
	}
	sugar := logger.Sugar()
	return sugar, nil
}
