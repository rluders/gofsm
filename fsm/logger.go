package fsm

import "log"

type DefaultLogger struct{}

func (l *DefaultLogger) Infof(format string, args ...any) {
	log.Printf("[INFO] "+format, args...)
}

func (l *DefaultLogger) Errorf(format string, args ...any) {
	log.Printf("[ERROR] "+format, args...)
}
