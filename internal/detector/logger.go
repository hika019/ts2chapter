package detector

import (
	"io"
	"log"
	"os"
)

var (
	Logger      *log.Logger
	DebugLogger *log.Logger
	debugMode   bool
)

func InitLogger(isDebug bool) {
	Logger = log.New(os.Stdout, "[INFO] ", log.LstdFlags)
	DebugLogger = log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
	if !isDebug {
		DebugLogger.SetOutput(io.Discard)
	}
}
