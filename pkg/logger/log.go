package logger

import (
	"log"
	"os"
)

var Logger = log.New(os.Stdout, "[APP] ", log.LstdFlags|log.Lshortfile)
