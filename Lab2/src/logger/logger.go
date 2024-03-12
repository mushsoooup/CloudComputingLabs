package logger

import "log"

var mode = "debug"

func Debug(format string, args ...any) {
	if mode == "debug" {
		log.Printf(format, args...)
	}
}
