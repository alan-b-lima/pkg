package opt

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"reflect"
	"strconv"
	"time"
)

func (o Opt[T]) Value() (driver.Value, error) {
	if !o.some {
		return null, nil // SQL null
	}

	return value(o.val)
}

func (o *Opt[T]) Scan(src any) error {
	if src == nil {
		o.some = false
		return nil
	}

	if err := scan(&o.val, src); err != nil {
		return err
	}

	o.some = true
	return nil
}

var null driver.Value = nil

func value(val any) (v driver.Value, rerr error) {
	if valuer, ok := val.(driver.Valuer); ok {
		return valuer.Value()
	}

	defer func() {
		var err error
		switch panic := recover().(type) {
		case nil:
			return // ignore
		case error:
			err = panic
		case string:
			err = errors.New(panic)
		default:
			err = errors.New(fmt.Sprint(panic))
		}

		rerr = fmt.Errorf("panicked: %w", err)
	}()

	rv := reflect.ValueOf(val)

	switch rv.Kind() {
	case reflect.Pointer:
		if rv.IsNil() {
			return null, nil
		}

		return value(rv.Elem().Interface())

	case reflect.Bool:
		return rv.Bool(), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u := rv.Uint()
		if u > math.MaxInt64 {
			return nil, fmt.Errorf("cannot parse %d as int64", u)
		}

		return int64(u), nil

	case reflect.Float32, reflect.Float64:
		return rv.Float(), nil

	case reflect.String:
		return rv.String(), nil

	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return rv.Bytes(), nil
		}

	case reflect.Struct:
		rt := rv.Type()
		if rt.PkgPath() == "time" && rt.Name() == "Time" {
			return rv.Interface(), nil
		}
	}

	return nil, fmt.Errorf("unsupported type: %T", val)
}

func scan(dst, src any) (rerr error) {
	if scanner, ok := any(dst).(sql.Scanner); ok {
		return scanner.Scan(src)
	}

	defer func() {
		var err error
		switch panic := recover().(type) {
		case nil:
			return // ignore
		case error:
			err = panic
		case string:
			err = errors.New(panic)
		default:
			err = errors.New(fmt.Sprint(panic))
		}

		rerr = fmt.Errorf("panicked: %w", err)
	}()

	rv := reflect.ValueOf(dst).Elem()

	switch src := src.(type) {
	case int64:
		return scan_int64(rv, src)
	case float64:
		return scan_float64(rv, src)
	case bool:
		return scan_bool(rv, src)
	case []byte:
		return scan_bytes(rv, src)
	case string:
		return scan_string(rv, src)
	case time.Time:
		return scan_time(rv, src)
	}

	return fmt.Errorf("unexpected type %v", rv.Type())
}

func scan_int64(dst reflect.Value, src int64) error {
	switch dst.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		if bits.Len64(uint64(src)) > dst.Type().Bits() {
			return fmt.Errorf("value %v unrepresentable as int", src)
		}

		dst.SetInt(src)
		return nil

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Uintptr:
		if src < 0 || bits.Len64(uint64(src)) > dst.Type().Bits() {
			return fmt.Errorf("value %v unrepresentable as %v", src, dst.Type())
		}

		dst.SetUint(uint64(src))
		return nil

	case reflect.Float32, reflect.Float64:
		dst.SetFloat(float64(src))
		return nil

	case reflect.String:
		dst.SetString(strconv.FormatInt(src, 10))
		return nil

	case reflect.Slice:
		if dst.Type().Elem().Kind() == reflect.Uint8 {
			dst.SetBytes([]byte(strconv.FormatInt(src, 10)))
			return nil
		}

	case reflect.Bool:
		dst.SetBool(src != 0)
		return nil
	}

	return fmt.Errorf("conversion from int64 to %v failed", dst.Type())
}

func scan_float64(dst reflect.Value, src float64) error {
	switch dst.Kind() {
	case reflect.Float32, reflect.Float64:
		dst.SetFloat(float64(src))
		return nil

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Uintptr:
		int, frac := math.Modf(src)
		if frac == 0 {
			return scan_int64(dst, int64(int))
		}

	case reflect.String:
		dst.SetString(strconv.FormatFloat(src, 10, -1, 64))
		return nil

	case reflect.Slice:
		if dst.Type().Elem().Kind() == reflect.Uint8 {
			dst.SetBytes([]byte(strconv.FormatFloat(src, 10, -1, 64)))
			return nil
		}
	}

	return fmt.Errorf("conversion from float64 to %v failed", dst.Type())
}

func scan_bool(dst reflect.Value, src bool) error {
	switch dst.Kind() {
	case reflect.Bool:
		dst.SetBool(src)
		return nil

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		if src {
			dst.SetInt(1)
		} else {
			dst.SetInt(0)
		}
		return nil

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Uintptr:
		if src {
			dst.SetUint(1)
		} else {
			dst.SetUint(0)
		}
		return nil

	case reflect.String:
		dst.SetString(strconv.FormatBool(src))
		return nil

	case reflect.Slice:
		if dst.Type().Elem().Kind() == reflect.Uint8 {
			dst.SetBytes([]byte(strconv.FormatBool(src)))
			return nil
		}
	}

	return fmt.Errorf("conversion from bool to %v failed", dst.Type())
}

func scan_bytes(dst reflect.Value, src []byte) error {
	err := scan_string(dst, string(src))
	if err != nil {
		return fmt.Errorf("conversion from []byte to %v failed", dst.Type())
	}

	return nil
}

func scan_string(dst reflect.Value, src string) error {
	switch dst.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(src)
		if err != nil {
			return fmt.Errorf("cannot parse %q as bool: %w", src, err)
		}

		dst.SetBool(b)
		return nil

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		i, err := strconv.ParseInt(src, 10, dst.Type().Bits())
		if err != nil {
			return fmt.Errorf("cannot parse %q as int: %w", src, err)
		}

		dst.SetInt(i)
		return nil

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Uintptr:
		u, err := strconv.ParseUint(src, 10, dst.Type().Bits())
		if err != nil {
			return fmt.Errorf("cannot parse %q as uint: %w", src, err)
		}

		dst.SetUint(u)
		return nil

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(src, dst.Type().Bits())
		if err != nil {
			return fmt.Errorf("cannot parse %q as float: %w", src, err)
		}

		dst.SetFloat(f)
		return nil

	case reflect.String:
		dst.SetString(src)
		return nil

	case reflect.Slice:
		if dst.Type().Elem().Kind() == reflect.Uint8 {
			dst.SetBytes([]byte(src))
			return nil
		}

	case reflect.Struct:
		rt := dst.Type()
		if rt.PkgPath() == "time" && rt.Name() == "Time" {
			t, err := time.Parse(time.RFC3339Nano, src)
			if err != nil {
				return fmt.Errorf("cannot parse %q as time: %w", src, err)
			}

			dst.Set(reflect.ValueOf(t))
			return nil
		}
	}

	return fmt.Errorf("conversion from string to %v failed", dst.Type())
}

func scan_time(dst reflect.Value, src time.Time) error {
	switch dst.Kind() {
	case reflect.Struct:
		rt := dst.Type()
		if rt.PkgPath() == "time" && rt.Name() == "Time" {
			dst.Set(reflect.ValueOf(src))
			return nil
		}

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Uintptr:
		return scan_int64(dst, src.UnixMilli())

	case reflect.String:
		dst.SetString(src.Format(time.RFC3339Nano))
		return nil

	case reflect.Slice:
		if dst.Type().Elem().Kind() == reflect.Uint8 {
			dst.SetBytes([]byte(src.Format(time.RFC3339Nano)))
			return nil
		}
	}

	return fmt.Errorf("conversion from time.Time to %v failed", dst.Type())
}
