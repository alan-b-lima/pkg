package problem

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Wrap wraps an error into a type that implements JSON marshaling and
// unmarshaling.
func Wrap(err error) error {
	if err == nil {
		return nil
	}

	return &wrapped{err}
}

type wrapped struct{ error }

var errUnmarshalable = errors.New("problem: failed to unmarshaled error into a sensible type")

// Wrap wraps an error into a type that implements JSON marshaling and
// unmarshaling.
func (e *wrapped) Error() string {
	return e.error.Error()
}

// String implements the [fmt.Stringer] interface on the type.
func (e *wrapped) String() string {
	return e.error.Error()
}

// Unwrap returns the wrapped error.
func (e *wrapped) Unwrap() error {
	return e.error
}

// MarshalJSON implements the [json.Marshaler] interface on the type.
func (e wrapped) MarshalJSON() ([]byte, error) {
	switch err := e.error.(type) {
	case *Multi:
		errs := make([]wrapped, 0, len(err.errs))
		for _, err := range err.errs {
			errs = append(errs, wrapped{err})
		}

		return json.Marshal(errs)

	case json.Marshaler:
		return err.MarshalJSON()

	case fmt.Stringer:
		return json.Marshal(err.String())
	}

	return json.Marshal(e.Error())
}

// UnmarshalJSON implements the [json.Unmarshaler] interface on the type. It
// tries to unmarshal the error into an [Error], if that fails, it tries to
// unmarshal it into a string, if that also fails, it returns an error.
func (e *wrapped) UnmarshalJSON(buf []byte) error {
	var err_ Error
	if err := json.Unmarshal(buf, &err_); err == nil {
		e.error = &err_
		return nil
	}

	var str_ string
	if err := json.Unmarshal(buf, &str_); err == nil {
		e.error = errors.New(str_)
		return nil
	}

	return errUnmarshalable
}

// Shadow wraps an error into a type that implements JSON marshaling and
// unmarshaling. However, unlike [Wrap], it hides internal information when
// marshaling. See [shadowed.MarshalJSON] for more info.
func Shadow(err error) error {
	if err == nil {
		return nil
	}

	return &shadowed{wrapped{err}}
}

type shadowed struct{ wrapped }

var errAllShadowed = errors.New("problem: all shadowed: nothing to display")

// MarshalJSON implements the [json.Marshaler] interface on the type. The term
// "visible" here refers to the JSON output only, [Shadow] affects nothing
// else.
//
// For the [*Error] type, the fields [Error.Kind], [Error.Title] and
// [Error.Message] are always visible.
// 
// For internal [*Error]s, the aforementioned field are the only visible
// fields, [Error.Cause] and [Error.Details] are hidden.
//
// For external [*Error]s, all fields are visible, with the added caveat that
// the [Error.Cause] field will be [Shadow]ed as well.
//
// For the [*Multi] type, they are marshalled as a JSON array of [Shadow]ed
// errors.
func (e shadowed) MarshalJSON() ([]byte, error) {
	switch err := e.error.(type) {
	case *Error:
		alt := Error{
			Kind:    err.Kind,
			Title:   err.Title,
			Message: err.Message,
		}

		if err.IsInternal() {
			return alt.MarshalJSON()
		}

		alt.Cause = Shadow(err.Cause)
		alt.Details = err.Details
		return alt.MarshalJSON()

	case *Multi:
		errs := make([]shadowed, 0, len(err.errs))
		for _, err := range err.errs {
			errs = append(errs, shadowed{wrapped{err}})
		}

		return json.Marshal(errs)
	}

	return nil, errAllShadowed
}
