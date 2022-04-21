package main

import (
	"fmt"
	"strings"
)

func main() {
	logLevelFlag := LogLevelFlag()
	formatFlag := NewFormatFlag()

	fmt.Printf("The level at which to log. Valid values are %s.\n", strings.Join(logLevelFlag.AllowedValues(), ", "))
	fmt.Printf("The format for log output. Valid values are %s.\n", strings.Join(formatFlag.AllowedValues(), ", "))

	logLevel := logLevelFlag.Parse()
	format := FormatJSON
	
	logger := DefaultLogger(logLevel, format)
	logger.Infof("Hello World")
}
