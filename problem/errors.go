// Package problem implements especialized functionalities on the
// domain of Go's error handling.
//
// This package constructs over the foundation of the [error]
// interface and [errors] package, as well as HTTP status codes and
// RFC 9457 about ploblem details (altough not complient to it).
package problem

import "encoding/json"

// Error is an structured error type.
type Error struct {
	Kind    Kind
	Title   string
	Message string
	Cause   error
	Details map[string]any
}

// New create a new error. Each new call generates a different error,
// regardless of the parameters.
func New(kind Kind, title, message string, cause error, details map[string]any) error {
	return &Error{
		Kind:    kind,
		Title:   title,
		Message: message,
		Cause:   cause,
		Details: details,
	}
}

// Error implements the [error] interface.
func (err *Error) Error() string {
	if err.Cause != nil {
		return err.Message + `: ` + err.Cause.Error()
	}

	return err.Message
}

// Unwrap returns the cause of the error, the cause might be nil.
func (err *Error) Unwrap() error {
	return err.Cause
}

// IsExternal identifies whether the error falls under the external
// category, see [Kind].
func (err *Error) IsExternal() bool {
	return err.Kind.IsExternal()
}

// IsCLient identifies whether the error falls under the internal
// category, see [Kind].
func (err *Error) IsInternal() bool {
	return err.Kind.IsInternal()
}

// MarshalJSON implements the [json.Marshaler] interface on the type,
// [Error.Cause] and [Error.Details] are ommited if nil.
func (err Error) MarshalJSON() ([]byte, error) {
	var efj errorForJSON
	efj = errorForJSON(err)

	if efj.Cause != nil {
		efj.Cause = &wrapped{efj.Cause}
	}

	return json.Marshal(efj)
}

// UnmarshalJSON implements the [json.Unmarshaler]
// interface on the type, might fail if the cause cannot be
// unmarshalled into a sensible error type.
func (err *Error) UnmarshalJSON(buf []byte) error {
	var efj errorFromJSON
	if err := json.Unmarshal(buf, &efj); err != nil {
		return err
	}

	*err = Error{
		Kind:    efj.Kind,
		Title:   efj.Title,
		Message: efj.Message,
		Cause:   &efj.Cause,
		Details: efj.Details,
	}
	return nil
}

type errorFromJSON struct {
	Kind    Kind           `json:"kind"`
	Title   string         `json:"title"`
	Message string         `json:"message"`
	Cause   wrapped        `json:"cause"`
	Details map[string]any `json:"metadata"`
}

type errorForJSON struct {
	Kind    Kind           `json:"kind"`
	Title   string         `json:"title"`
	Message string         `json:"message"`
	Cause   error          `json:"cause,omitempty"`
	Details map[string]any `json:"metadata,omitempty"`
}
