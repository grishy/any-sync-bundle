package lightfilenodestore

import (
	"fmt"
)

// badgerLogger implements badger.Logger interface to redirect BadgerDB logs
// to our application logger
type badgerLogger struct{}

func (b badgerLogger) Errorf(s string, i ...any) {
	log.Error("badger: " + fmt.Sprintf(s, i...))
}

func (b badgerLogger) Warningf(s string, i ...any) {
	log.Warn("badger: " + fmt.Sprintf(s, i...))
}

func (b badgerLogger) Infof(s string, i ...any) {
	log.Info("badger: " + fmt.Sprintf(s, i...))
}

func (b badgerLogger) Debugf(s string, i ...any) {
	log.Debug("badger: " + fmt.Sprintf(s, i...))
}
