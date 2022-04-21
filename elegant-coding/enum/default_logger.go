package main

import (
	"github.com/sirupsen/logrus"
	"os"
)

// DefaultLogger returns a Logger with the default properties.
// The desired output format is passed as a LogFormat Enum.
func DefaultLogger(level logrus.Level, format Format) *logrus.Logger {
	logger := logrus.New()

	if format == FormatJSON {
		logger.Formatter = new(logrus.JSONFormatter)
	}

	// Make sure the output is set to stdout so log messages don't show up as errors in cloud log dashboards.
	logger.Out = os.Stdout

	logger.Level = level

	return logger
}
