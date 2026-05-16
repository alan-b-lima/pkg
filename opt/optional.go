// Package opt implements a optional type, it may or may not contain
// a value.
package opt

import "fmt"

// Opt is an optional type, it may or may not contatin a value of a
// certain type, given by its type parameter. Its zero value
// indicates the absence of value, when printed: <none>.
//
// An arbitrary [Opt] can be marshalled and unmarshalled with JSON,
// with JSON's null being considered the zero value, regardless of
// the type param. If available, it will try to use [json.Marshaler]
// and [json.Unmarshaler] for the type param.
//
// An arbitrary [Opt] can be valued and scanned with SQL, with SQL's
// null being considered the zero value, regardless of the type
// param. If available, it will try to use [driver.Valuer] and
// [sql.Scanner] for the type param.
type Opt[T any] struct {
	val  T
	some bool
}

// Some creates a new optional with the given value.
func Some[T any](val T) Opt[T] {
	return Opt[T]{val, true}
}

// None creates a new optional with no value.
func None[T any]() Opt[T] {
	return Opt[T]{}
}

// Unwrap unpacks the Opt and returns its components, a common
// assertion may be done:
//
//	val, ok := opt.Unwrap()
//	if !ok {
//		// handle noneness
//	}
//
// If ok is false, val will be the zero value of T.
func (o Opt[T]) Unwrap() (val T, ok bool) {
	if o.some {
		return o.val, true
	}
	return
}

// Present reports whether the optional has a value.
func (o Opt[T]) Present() bool {
	return o.some
}

// IsZero reports whether the optional does not have a value.
func (o Opt[T]) IsZero() bool {
	return !o.some
}

// Or returns the value of the optional if it is present, otherwise
// it returns the given default value.
func (o Opt[T]) Or(def T) T {
	if o.some {
		return o.val
	}
	return def
}

// Interface returns the value of the optional as an empty interface,
// if it is present, otherwise it returns nil.
//
// If the optional is of an interface type, the nil by absence and
// a present nil are indistinguishable.
func (o Opt[T]) Interface() any {
	if o.some {
		return o.val
	}
	return nil
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
