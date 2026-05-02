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
	ErrNotPointerToStruct = errors.New("query: v must be a pointer to struct type")
	ErrNilPointer         = errors.New("query: v is a nil pointer to struct")

	ErrPanic         = errors.New("query: panic while parsing")
	ErrUnsettable    = errors.New("query: cannot change contents of v")
	ErrUnaddressable = errors.New("query: field is not addressable")

	ErrUnsupportedType = errors.New("query: unsupported type")
	ErrConvertion      = errors.New("query: convertion error")
)

func Parse(q url.Values, v any) (err error) {
	defer func() {
		if recover() != nil {
			err = ErrPanic
		}
	}()

	if v == nil {
		return ErrNilPointer
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

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		num, err := strconv.ParseInt(val[0], 10, rtype.Bits())
		if err != nil {
			return conerr(rtype, err)
		}

		rvalue.SetInt(int64(num))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		num, err := strconv.ParseUint(val[0], 10, rtype.Bits())
		if err != nil {
			return conerr(rtype, err)
		}

		rvalue.SetUint(uint64(num))

	case reflect.Float32, reflect.Float64:
		num, err := strconv.ParseFloat(val[0], rtype.Bits())
		if err != nil {
			return conerr(rtype, err)
		}

		rvalue.SetFloat(num)

	case reflect.Complex64, reflect.Complex128:
		num, err := strconv.ParseComplex(val[0], rtype.Bits())
		if err != nil {
			return conerr(rtype, err)
		}

		rvalue.SetComplex(num)

	case reflect.Bool:
		b, err := strconv.ParseBool(val[0])
		if err != nil {
			return conerr(rtype, err)
		}

		rvalue.SetBool(b)

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

	case reflect.Struct:
		if !(rtype.PkgPath() == "time" && rtype.Name() == "Time") {
			return ErrUnsupportedType
		}

		t, err := time.Parse(time.RFC3339Nano, val[0])
		if err != nil {
			return conerr(rtype, err)
		}

		rvalue.Set(reflect.ValueOf(t))

	default:
		return ErrUnsupportedType
	}

	return nil
}

type ConvertionError struct {
	Type reflect.Type
	Err  error
}

func (e *ConvertionError) Error() string {
	return fmt.Sprintf("query: not convertible to %v: %v", e.Type, e.Err)
}

func (e *ConvertionError) Unwrap() []error {
	return []error{ErrConvertion, e.Err}
}

func conerr(t reflect.Type, err error) error {
	return &ConvertionError{Type: t, Err: err}
}
