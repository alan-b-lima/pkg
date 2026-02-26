// Package opt implements a optional type, it may or may not contain
// a value. It can also be marshalled and unmarshalled with json,
// using the undelying marshelers, if such exist.
package opt

import "fmt"

// Opt is an optional type, it may or may not contatin a value of a
// certain type, given by its type parameter. Its zero value is
// called None, when printed: <none>, it indicates the absence of
// value.
//
// As it stands, an arbitrary [Opt] can be marshalled and
// unmarshalled with JSON, with JSON's null being considered the zero
// value, regardless of the type param.
type Opt[T any] struct {
	val  T
	some bool
}

// Some creates a new Opt value with a value.
func Some[T any](val T) Opt[T] {
	return Opt[T]{val, true}
}

// None creates a new Opt value with no value, ie, None.
func None[T any]() Opt[T] {
	return Opt[T]{}
}

// Unwrap unpacks the Opt struct and returns its components, a common
// assertion should be done, it may be done in the following way:
//
//	val, ok := opt.Unwrap()
//	if !ok {
//		// handle noneness
//	}
//
// Since this a non-build-in function, the ok return cannot be
// ommited.
func (o Opt[T]) Unwrap() (T, bool) {
	return o.val, o.some
}

// String implements the [fmt.Stringer] interface, it returns the
// string "<none>" if the Opt is None, otherwise it returns the
// string representation of the underlying value, using [fmt.Sprint].
func (o Opt[T]) String() string {
	if !o.some {
		return "<none>"
	}

	return fmt.Sprint(o.val)
}
