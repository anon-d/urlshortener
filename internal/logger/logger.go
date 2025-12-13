package logger

import (
	"log"

	"go.uber.org/zap"
)

type Logger struct {
	ZLog *zap.SugaredLogger
}

func New() (*Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Printf("Init logger: error %v", err)
		return &Logger{}, err
	}
	sugar := logger.Sugar()
	return &Logger{ZLog: sugar}, nil
}
