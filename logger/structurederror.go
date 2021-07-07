package logger

import (
	"strings"

	"github.com/relex/gotils/logger/priv"
	"github.com/sirupsen/logrus"
)

// getMergedEntryFromArgs scans the given arguments and make merged logger from the first StructuredError if present
func getMergedEntryFromArgs(parent *logrus.Entry, args []interface{}) *logrus.Entry {
	for i, a := range args {
		if serr, ok := a.(*StructuredError); ok {
			args[i] = serr.Unwrap()
			return serr.getEntry(parent)
		}
	}

	return parent
}

// StructuredError represents a thing that carries metadata that should be elevated to log fields when logged
type StructuredError struct {
	fields map[string]interface{}
	err    error
}

// NewStructuredError creates a StructuredError with a map of fields (to be copied) and a message
func NewStructuredError(srcFields map[string]interface{}, err error) *StructuredError {
	newFields := make(map[string]interface{}, len(srcFields))
	for k, v := range srcFields {
		if k == priv.LabelComponent {
			k = "errorComponent"
		}
		newFields[k] = v
	}

	return &StructuredError{
		fields: newFields,
		err:    err,
	}
}

func (se *StructuredError) Error() string {
	return se.String()
}

func (se *StructuredError) String() string {
	strList := buildSprintPrefixes(se.fields)
	if se.err != nil {
		strList = append(strList, se.err.Error())
	}
	return strings.Join(strList, " ")
}

func (se *StructuredError) Unwrap() error {
	return se.err
}

func (se *StructuredError) getEntry(parent *logrus.Entry) *logrus.Entry {
	if len(se.fields) == 0 {
		return parent
	}

	return parent.WithFields(se.fields)
}
