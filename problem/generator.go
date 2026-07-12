package problem

import "fmt"

// ImpError is an error implementation that can be incrementally constructed,
// up until final construction, through [ImpError.Make], all [ImpError]s are
// passed around as values, therefore, all previous [ImpError]s are oblivious
// to future changes.
type ImpError struct {
	kind    Kind
	title   string
	message string
	cause   error
	details map[string]any
}

// Imp initiates a new error implementation.
func Imp(kind Kind, title string) ImpError {
	return ImpError{
		kind:  kind,
		title: title,
	}
}

// Message replaces the error message of the implementation.
func (b ImpError) Message(message string) ImpError {
	b.message = message
	return b
}

// Message replaces the error message with a formatted message.
func (b ImpError) Messagef(format string, args ...any) ImpError {
	b.message = fmt.Sprintf(format, args...)
	return b
}

// Format initiates a new error implementation formatter with a predefined
// format.
func (b ImpError) Format(format string) FmtError {
	return FmtError{
		kind:    b.kind,
		title:   b.title,
		format:  format,
		cause:   b.cause,
		details: b.details,
	}
}

// Cause replaces the error cause of the implementation.
func (b ImpError) Cause(cause error) ImpError {
	b.cause = cause
	return b
}

// Details replaces the error details of the implementation.
func (b ImpError) Details(details map[string]any) ImpError {
	b.details = details
	return b
}

// Make constructs a new error out of the current values in the fields of the
// implementation.
func (b ImpError) Make() error {
	return New(b.kind, b.title, b.message, b.cause, b.details)
}

// Error implement the [error] interface.
//
// This type implements the error interface to allow values to be used as
// targets of [errors.Is].
func (b *ImpError) Error() string {
	return b.Make().Error()
}

// FmtError is an error implementation that can be incrementally constructed,
// up until final construction, value semantics are [FmtError] the same as in
// [ImpError].
type FmtError struct {
	kind    Kind
	title   string
	format  string
	cause   error
	details map[string]any
}

// Cause replaces the error cause of the implementation.
func (gen FmtError) Cause(cause error) FmtError {
	gen.cause = cause
	return gen
}

// Details replaces the error details of the implementation.
func (gen FmtError) Details(details map[string]any) FmtError {
	gen.details = details
	return gen
}

// Make constructs a new error out of the current values in the fields of the
// implementation.
func (gen FmtError) Make(a ...any) error {
	message := fmt.Sprintf(gen.format, a...)
	return New(gen.kind, gen.title, message, gen.cause, gen.details)
}

// Error implement the [error] interface.
//
// This type implements the error interface to allow values to be used as
// targets of [errors.Is].
func (gen *FmtError) Error() string {
	return gen.Make().Error()
}
