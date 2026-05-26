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

// Error implements the [error] interface on the type.
func (e *wrapped) Error() string {
	return e.error.Error()
}

// String implements the [fmt.Stringer] interface on the type.
func (e *wrapped) String() string {
	return e.Error()
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

// UnmarshalJSON implements the [json.Unmarshaler] interface on the type. The
// unmarshaler tries to unmarshal in the following order:
//
//   - into a [string], to be given to [errors.New].
//   - into an *[Error].
//   - into a [Multi].
//
// If none of this works, this function returns an error.
func (e *wrapped) UnmarshalJSON(buf []byte) error {
	var str string
	if err := json.Unmarshal(buf, &str); err == nil {
		e.error = errors.New(str)
		return nil
	}

	var error Error
	if err := json.Unmarshal(buf, &error); err == nil {
		e.error = &error
		return nil
	}

	var multi Multi
	if err := json.Unmarshal(buf, &multi); err == nil {
		e.error = &multi
		return nil
	}

	return errUnmarshalable
}

// Shadow wraps an error into a type that implements JSON marshaling and
// unmarshaling. However, unlike [Wrap], it hides internal information when
// marshaling to JSON.
//
// In all aspects besides JSON marshaling, a [Shadow]ed error is equivalent to
// a [Wrap]ed error.
func Shadow(err error) error {
	if err == nil {
		return nil
	}

	return &shadow{err}
}

type shadow struct{ error }

// Error implements the [error] interface on the type.
func (e *shadow) Error() string {
	return e.error.Error()
}

// String implements the [fmt.Stringer] interface on the type.
func (e *shadow) String() string {
	return e.Error()
}

// Unwrap returns the shadow error.
func (e *shadow) Unwrap() error {
	return e.error
}

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
//
// For all other errors, the string "opaque", including the quotes, is returned
// as a byte slice.
func (e shadow) MarshalJSON() ([]byte, error) {
	switch err := e.error.(type) {
	case *Error:
		if err.IsInternal() {
			alt := Error{
				Kind:    err.Kind,
				Title:   err.Title,
				Message: err.Message,
			}

			return alt.MarshalJSON()
		} else {
			alt := Error{
				Kind:    err.Kind,
				Title:   err.Title,
				Message: err.Message,
				Cause:   Shadow(err.Cause),
				Details: err.Details,
			}

			return alt.MarshalJSON()
		}

	case *Multi:
		errs := make([]shadow, 0, len(err.errs))
		for _, err := range err.errs {
			errs = append(errs, shadow{err})
		}

		return json.Marshal(errs)
	}

	return []byte(`"opaque"`), nil
}

func (e *shadow) UnmarshalJSON(buf []byte) error {
	var wrp wrapped
	if err := wrp.UnmarshalJSON(buf); err != nil {
		return err
	}

	*e = shadow(wrp)
	return nil
}
