package main

import (
	"github.com/pkg/errors"
)

// Enum is a Cobra-compatible wrapper for defining
// a string flag that can be one of a specified set
// of values.
type Enum struct {
	allowedValues []string
	value         string
}

// NewEnum returns a new enum flag with the specified list
// of allowed values, and the specified default value if
// none is set.
func NewEnum(defaultValue string, allowedValues ...string) *Enum {
	return &Enum{
		allowedValues: allowedValues,
		value:         defaultValue,
	}
}

// String returns a string representation of the
// enum flag.
func (e *Enum) String() string {
	return e.value
}

// Set assigns the provided string to the enum
// receiver. It returns an error if the string
// is not an allowed value.
func (e *Enum) Set(s string) error {
	for _, val := range e.allowedValues {
		if val == s {
			e.value = s
			return nil
		}
	}

	return errors.Errorf("invalid value: %q", s)
}

// Type returns a string representation of the Enum type.
func (e *Enum) Type() string {
	// we don't want the help text to display anything regarding
	// the type because the usage text for the flag should capture
	// the possible options.
	return ""
}

// AllowedValues returns a slice of the flag's valid
// values.
func (e *Enum) AllowedValues() []string {
	return e.allowedValues
}
