package opt

import (
	"bytes"
	"encoding/json"
)

const jsonNull = `null`

var jsonNullBytes = []byte(jsonNull)

// MarshalJSON implements the [json.Marshaler] interface, it marshals
// the Opt struct, if it is None, it returns JSON's null, otherwise
// it tries to marshal the underlying value, if it fails, the error
// is returned.
func (o Opt[T]) MarshalJSON() ([]byte, error) {
	if !o.some {
		return []byte(jsonNull), nil
	}

	return json.Marshal(o.val)
}

// UnmarshalJSON implements the [json.Unmarshaler] interface, it
// unmarshals the value into the Opt struct, if the input is JSON's
// null, the Opt is set to None, otherwise it tries to unmarshal the
// value into the underlying type, if it fails, the Opt is set to
// None and the error is returned.
func (o *Opt[T]) UnmarshalJSON(b []byte) error {
	o.some = !bytes.Equal(jsonNullBytes, b)
	if !o.some {
		return nil
	}

	if err := json.Unmarshal(b, &o.val); err != nil {
		*o = Opt[T]{}
		return err
	}

	return nil
}
