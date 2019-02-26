package logger

import (
	"log"
)

var logger *log.Logger

func SetDefault(l *log.Logger) {
	logger = l
}

func Info(args ...interface{}) {
	logger.Println(args...)
}

func Infof(format string, args ...interface{}) {
	logger.Printf(format, args...)
}
