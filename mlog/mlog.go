package mlog

import (
	"log"
	"os"
)

// Debug level
const (
	INFO = iota
	DEBUG
	WARNING
	ERROR
)

// Mlog my manual log
type Mlog struct {
	logLevel byte
	info     *log.Logger
	debug    *log.Logger
	warning  *log.Logger
	err      *log.Logger
}

// Mlog My log
var mlog *Mlog

// SetMinLevel set min debug level
func SetMinLevel(level byte) {
	mlog.logLevel = level
}

// Info print info log
func Info(v ...interface{}) {
	if INFO >= mlog.logLevel {
		mlog.info.Println(v...)
	}
}

// Debug print debug log
func Debug(v ...interface{}) {
	if DEBUG >= mlog.logLevel {
		mlog.debug.Println(v...)
	}
}

// Warning print warning log
func Warning(v ...interface{}) {
	if WARNING >= mlog.logLevel {
		mlog.warning.Println(v...)
	}
}

// Error print error log
func Error(v ...interface{}) {
	if ERROR >= mlog.logLevel {
		mlog.err.Println(v...)
	}
}

func init() {
	mlog = &Mlog{
		logLevel: INFO,
		info:     log.New(os.Stdout, "[I] ", log.Ldate|log.Ltime),
		debug:    log.New(os.Stdout, "[D] ", log.Ldate|log.Ltime),
		warning:  log.New(os.Stdout, "[W] ", log.Ldate|log.Ltime|log.Lshortfile),
		err:      log.New(os.Stdout, "[E] ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}
