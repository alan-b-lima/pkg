package problem

import "fmt"

type imp struct {
	kind     Kind
	title    string
	message  string
	cause    error
	details map[string]any
}

// Imp is an error implementation that can be incrementally constructed, up
// until final construction, through [imp.Make], all [imp]s are passed around
// as values, therefore, an assignment will only affect those after it.
func Imp(kind Kind, title string) imp {
	return imp{
		kind:  kind,
		title: title,
	}
}

// Message replaces the error message of the implementation.
func (b imp) Message(message string) imp {
	b.message = message
	return b
}

// Message replaces the error message of the implementation.
func (b imp) Format(format string) fmt_ {
	return fmt_{
		kind:    b.kind,
		title:   b.title,
		format:  format,
		cause:   b.cause,
		details: b.details,
	}
}

// Cause replaces the error cause of the implementation.
func (b imp) Cause(cause error) imp {
	b.cause = cause
	return b
}

// Metadata replaces the error metadata of the implementation.
func (b imp) Metadata(details map[string]any) imp {
	b.details = details
	return b
}

// Make constructs a new error out of the current values in the fields of the
// implementation. Every call to Make creates a different error.
func (b imp) Make() error {
	return New(b.kind, b.title, b.message, b.cause, b.details)
}

type fmt_ struct {
	kind    Kind
	title   string
	format  string
	cause   error
	details map[string]any
}

// Cause replaces the error cause of the implementation.
func (gen fmt_) Cause(cause error) fmt_ {
	gen.cause = cause
	return gen
}

// Metadata replaces the error metadata of the implementation.
func (gen fmt_) Metadata(metadata map[string]any) fmt_ {
	gen.details = metadata
	return gen
}

func (gen fmt_) Make(a ...any) error {
	message := fmt.Errorf(gen.format, a...).Error()
	return New(gen.kind, gen.title, message, gen.cause, gen.details)
}
