package main

// Format is a string representation of the desired output format for logs
type Format string

const (
	FormatText   Format = "text"
	FormatJSON   Format = "json"
	defaultValue Format = FormatText
)

// FormatFlag is a command-line flag for setting the logrus
// log format.
type FormatFlag struct {
	*Enum
	defaultValue Format
}

// NewFormatFlag constructs a new log level flag.
func NewFormatFlag() *FormatFlag {
	return &FormatFlag{
		Enum: NewEnum(
			string(defaultValue),
			string(FormatText),
			string(FormatJSON),
		),
		defaultValue: defaultValue,
	}
}

// Parse returns the flag's value as a Format.
func (f *FormatFlag) Parse() Format {
	return Format(f.String())
}
