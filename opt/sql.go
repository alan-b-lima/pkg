package opt

import (
	"database/sql"
	"database/sql/driver"
)

func (o Opt[T]) Value() (driver.Value, error) {
	s := sql.Null[T]{
		V:     o.val,
		Valid: o.some,
	}

	return s.Value()
}

func (o *Opt[T]) Scan(src any) error {
	var s sql.Null[T]
	if err := s.Scan(src); err != nil {
		return err
	}

	*o = Opt[T]{
		val:  s.V,
		some: s.Valid,
	}
	return nil
}
