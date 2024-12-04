package logger

import (
	"fmt"
	"log"
	"os"

	"FMgo/internal/config"
)

var (
	logger *log.Logger
	file   *os.File
)

func Init() error {
	var err error
	file, err = os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	logger = log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)
	Info("Logger initialized")
	return nil
}

func Close() {
	if logger != nil {
		Info("Logger closing")
	}
	if file != nil {
		file.Close()
	}
}

func Info(format string, v ...interface{}) {
	if logger != nil {
		logger.Output(2, fmt.Sprintf("[INFO] "+format, v...))
	}
}

func Error(format string, v ...interface{}) {
	if logger != nil {
		logger.Output(2, fmt.Sprintf("[ERROR] "+format, v...))
	}
}

func Debug(format string, v ...interface{}) {
	if logger != nil {
		logger.Output(2, fmt.Sprintf("[DEBUG] "+format, v...))
	}
}
