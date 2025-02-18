package lightfilenodestore

import (
	"fmt"
)

// badgerLogger implements badger.Logger interface to redirect BadgerDB logs
// to our application logger with appropriate prefixes
type badgerLogger struct{}

func (b badgerLogger) Errorf(s string, i ...interface{}) {
	log.Error("badger: " + fmt.Sprintf(s, i...))
}

func (b badgerLogger) Warningf(s string, i ...interface{}) {
	log.Warn("badger: " + fmt.Sprintf(s, i...))
}

func (b badgerLogger) Infof(s string, i ...interface{}) {
	log.Info("badger: " + fmt.Sprintf(s, i...))
}

func (b badgerLogger) Debugf(s string, i ...interface{}) {
	log.Debug("badger: " + fmt.Sprintf(s, i...))
}
