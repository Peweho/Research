package util

import (
	"log"
	"os"
)

var Log = log.New(os.Stdout, "[Research]", log.Lshortfile|log.Ldate|log.Ltime)
