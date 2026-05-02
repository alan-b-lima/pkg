// Package query provides a function to parse search query parameters from
// [url.Values] into a struct.
package query

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"time"
)

var (
	ErrPanic              = errors.New("query: panic while parsing")
	ErrNotPointerToStruct = errors.New("query: v must be a pointer to struct type")

	ErrUnsettable    = errors.New("query: cannot change contents of v")
	ErrUnaddressable = errors.New("query: field is not addressable")

	ErrUnsupportedType = errors.New("query: unsupported type")
)

// Unmarshal parses the search query parameters from url.Values into the struct
// into the value pointed to by v. If v is nil or not a pointer, Unmarshal
// returns an [ErrNotPointerToStruct].
//
// The struct fields must be exported and tagged with `query:"name"` to be
// parsed, name is the key in url.Values.
//
// Unmarshal supports the following types:
//
//	string
//	int, int8, int16, int32, int64
//	uint, uint8, uint16, uint32, uint64, uintptr
//	float32, float64
//	complex64, complex128
//	bool
//	slices of the above types
//	time.Time (in RFC3339Nano format)
//
// If a unsupported type is given, [ErrUnsupportedType] us reported. If any
// convertion fails, a [ConvertionError] is reported.
//
// If a name appears multiple times in the query params, like
// “?name=1&name=2”, for all types but slices only the first will be
// considered. For slices, each element is parsed as its underlying type an put
// in the same order they appear in the query.
//
// Bytes slices are NOT treated as special.
func Unmarshal(q url.Values, v any) (err error) {
	defer func() {
		if panic := recover(); panic != nil {
			err = ErrPanic
			if e, ok := panic.(error); ok {
				err = fmt.Errorf("%w: %w", ErrPanic, e)
			}
		}
	}()

	if v == nil {
		return ErrNotPointerToStruct
	}

	rvalue := reflect.ValueOf(v)

	if rvalue.Kind() != reflect.Pointer {
		return ErrNotPointerToStruct
	}

	if rvalue.Elem().Kind() != reflect.Struct {
		return ErrNotPointerToStruct
	}

	return query_struct(q, v)
}

func query_struct(q url.Values, v any) error {
	rtype := reflect.TypeOf(v).Elem()
	rvalue := reflect.ValueOf(v).Elem()

	if !rvalue.CanSet() {
		return ErrUnsettable
	}

	for i := range rtype.NumField() {
		rsfield := rtype.Field(i)
		if !rsfield.IsExported() {
			continue
		}

		query := rsfield.Tag.Get("query")
		val, in := q[query]
		if !in || len(val) == 0 {
			continue
		}

		field := rvalue.Field(i)
		if !field.CanAddr() {
			return ErrUnaddressable
		}
		if err := query_var(val, field.Addr().Interface()); err != nil {
			return err
		}
	}

	return nil
}

func query_var(val []string, v any) error {
	rtype := reflect.TypeOf(v).Elem()
	rvalue := reflect.ValueOf(v).Elem()

	if !rvalue.CanSet() {
		return ErrUnsettable
	}

	switch rtype.Kind() {
	case reflect.String:
		rvalue.SetString(val[0])
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		num, err := strconv.ParseInt(val[0], 10, rtype.Bits())
		if err != nil {
			return converr(rtype, err)
		}

		rvalue.SetInt(int64(num))
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		num, err := strconv.ParseUint(val[0], 10, rtype.Bits())
		if err != nil {
			return converr(rtype, err)
		}

		rvalue.SetUint(uint64(num))
		return nil

	case reflect.Float32, reflect.Float64:
		num, err := strconv.ParseFloat(val[0], rtype.Bits())
		if err != nil {
			return converr(rtype, err)
		}

		rvalue.SetFloat(num)
		return nil

	case reflect.Complex64, reflect.Complex128:
		num, err := strconv.ParseComplex(val[0], rtype.Bits())
		if err != nil {
			return converr(rtype, err)
		}

		rvalue.SetComplex(num)
		return nil

	case reflect.Bool:
		b, err := strconv.ParseBool(val[0])
		if err != nil {
			return converr(rtype, err)
		}

		rvalue.SetBool(b)
		return nil

	case reflect.Slice:
		if rtype.Elem().Kind() == reflect.String {
			rvalue.Set(reflect.ValueOf(val))
			return nil
		}

		var nvalue reflect.Value
		if len(val) <= rvalue.Cap() {
			nvalue = rvalue.Slice(0, len(val))
		} else {
			nvalue = reflect.MakeSlice(rtype, len(val), len(val))
		}
		rvalue.Set(nvalue)

		for i := range rvalue.Len() {
			cell := rvalue.Index(i)
			if !cell.CanAddr() {
				return ErrUnaddressable
			}

			if err := query_var(val[i:i+1], cell.Addr().Interface()); err != nil {
				return err
			}
		}

		return nil

	case reflect.Struct:
		if rtype.PkgPath() == "time" && rtype.Name() == "Time" {
			t, err := time.Parse(time.RFC3339Nano, val[0])
			if err != nil {
				return converr(rtype, err)
			}

			rvalue.Set(reflect.ValueOf(t))
			return nil
		}
	}

	return ErrUnsupportedType
}

// ConvertionError is an error that occurs when a value cannot be converted to
// the target type. It contains the target type and the underlying error that
// caused the failure.
type ConvertionError struct {
	Type reflect.Type
	Err  error
}

func converr(t reflect.Type, err error) error {
	return &ConvertionError{Type: t, Err: err}
}

func (e *ConvertionError) Error() string {
	return fmt.Sprintf("query: not convertible to %v: %v", e.Type, e.Err)
}

func (e *ConvertionError) Unwrap() error {
	return e.Err
}
