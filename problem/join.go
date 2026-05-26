package problem

import (
	"encoding/json"
	"strings"
)

// Multi is an error that represents multiple errors.
type Multi struct{ errs []error }

// Join combines multiple errors into a single error. Nil errors are ignored.
// If all errors are nil, Join returns nil.
func Join(errs ...error) error {
	var n int
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}

	merr := Multi{errs: make([]error, 0, n)}
	for _, err := range errs {
		if err != nil {
			merr.errs = append(merr.errs, err)
		}
	}

	return &merr
}

// Error implements the [error] interface.
func (e *Multi) Error() string {
	if e == nil || len(e.errs) == 0 {
		return "<nil>"
	}

	if len(e.errs) == 1 {
		return e.errs[0].Error()
	}

	var b strings.Builder
	b.WriteString(e.errs[0].Error())
	for _, err := range e.errs[1:] {
		b.WriteByte('\n')
		b.WriteString(err.Error())
	}

	return b.String()
}

// Unwrap implements the interface for unwrapping errors.
func (e *Multi) Unwrap() []error {
	return e.errs
}

// MarshalJSON implements the [json.Marshaler] interface on the type.
func (e Multi) MarshalJSON() ([]byte, error) {
	wrps := make([]wrapped, 0, len(e.errs))
	for _, err := range e.errs {
		wrps = append(wrps, wrapped{err})
	}

	return json.Marshal(e.errs)
}

// UnmarshalJSON implements the [json.Unmarshaler] interface on the type.
func (e *Multi) UnmarshalJSON(buf []byte) error {
	var wrps []wrapped
	if err := json.Unmarshal(buf, &wrps); err != nil {
		return err
	}

	errs := make([]error, 0, len(wrps))
	for _, err := range wrps {
		errs = append(errs, &err)
	}

	if err := Join(errs...); err != nil {
		*e = *err.(*Multi)
		return nil
	}

	*e = Multi{}
	return nil
}
