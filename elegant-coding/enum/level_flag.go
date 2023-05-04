package main

import (
	"github.com/sirupsen/logrus"
	"sort"
	"strings"
)

var sortedLogLevels = sortLogLevels()

// LevelFlag is a command-line flag for setting the logrus
// log level.
type LevelFlag struct {
	*Enum
	defaultValue logrus.Level
}

// LogLevelFlag constructs a new log level flag.
func LogLevelFlag() *LevelFlag {
	return &LevelFlag{
		Enum:         NewEnum(logrus.InfoLevel.String(), sortedLogLevels...),
		defaultValue: logrus.InfoLevel,
	}
}

// Parse returns the flag's value as a logrus.Level.
func (f *LevelFlag) Parse() logrus.Level {
	if parsed, err := logrus.ParseLevel(f.String()); err == nil {
		return parsed
	}

	// This should theoretically never happen assuming the enum flag
	// is constructed correctly because the enum flag will not allow
	//  an invalid value to be set.
	logrus.Errorf("log-level flag has invalid value %s", strings.ToUpper(f.String()))
	return f.defaultValue
}

// sortLogLevels returns a string slice containing all of the valid logrus
// log levels (based on logrus.AllLevels), sorted in ascending order of severity.
func sortLogLevels() []string {
	var (
		sortedLogLevels  = make([]logrus.Level, len(logrus.AllLevels))
		logLevelsStrings []string
	)

	copy(sortedLogLevels, logrus.AllLevels)

	// logrus.Panic has the lowest value, so the compare function uses ">"
	sort.Slice(sortedLogLevels, func(i, j int) bool { return sortedLogLevels[i] > sortedLogLevels[j] })

	for _, level := range sortedLogLevels {
		logLevelsStrings = append(logLevelsStrings, level.String())
	}

	return logLevelsStrings
}
