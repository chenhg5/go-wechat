package main

import (
	"io"
	"fmt"
	"os"
)

type Logger struct {
	Writer io.Writer
}

func (logger *Logger) log(a ...interface{}) {
	fmt.Fprintf((*logger).Writer, "%s", a...)
}

var AccessLogger *Logger
var ErrorLogger *Logger


func InitLogger()  {
	f, _ := os.Create(EnvConfig["ACCESS_LOG_PATH"].(string))
	defer f.Close()

	(*AccessLogger).Writer = io.MultiWriter(f)

	f, _ = os.Create(EnvConfig["ERROR_LOG_PATH"].(string))
	defer f.Close()

	(*ErrorLogger).Writer = io.MultiWriter(f)
}