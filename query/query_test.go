package query_test

import (
	"errors"
	"net/url"
	"reflect"
	"testing"
	"time"

	. "github.com/alan-b-lima/pkg/query"
)

func TestRejectsInvalidInput(t *testing.T) {
	type Test struct {
		Name  string
		Input any
		Want  error
	}

	tests := []Test{
		{Name: "nil-pointer", Input: nil, Want: ErrNilPointer},
		{Name: "non-pointer", Input: struct{}{}, Want: ErrNotPointerToStruct},
		{Name: "pointer to non-struct", Input: new(int), Want: ErrNotPointerToStruct},
		{Name: "nil pointer to struct", Input: (*struct{})(nil), Want: ErrNotPointerToStruct},
	}

	for _, test := range tests {
		err := Parse(url.Values{"name": {"alice"}}, test.Input)
		if err == nil {
			t.Errorf("query test %+q shouldn't have succeded", test.Name)
			continue
		}

		if !errors.Is(err, test.Want) {
			t.Errorf("%+q: %v", test.Name, err)
		}
	}
}

func TestParsing(t *testing.T) {
	type Input struct {
		Name     string     `query:"name"`
		Age      int        `query:"age"`
		Balance  uint32     `query:"balance"`
		Hidden   bool       `query:"hidden"`
		Tags     []string   `query:"tag"`
		Numbers  []int      `query:"num"`
		Money    float64    `query:"money"`
		Position complex128 `query:"position"`
		Created  time.Time  `query:"created"`
	}

	type Test struct {
		given    url.Values
		expected Input
	}

	timestamp := time.Date(2026, time.March, 14, 15, 92, 65, 0, time.UTC)

	tests := []Test{
		{
			given: url.Values{
				"name":     {"Luan"},
				"age":      {"23"},
				"balance":  {"900"},
				"hidden":   {"false"},
				"tag":      {"admin", "ops"},
				"num":      {"7", "11", "13"},
				"money":    {"123.45"},
				"position": {"1+2i"},
				"created":  {timestamp.Format(time.RFC3339Nano)},
			},
			expected: Input{
				Name:     "Luan",
				Age:      23,
				Balance:  900,
				Hidden:   false,
				Tags:     []string{"admin", "ops"},
				Numbers:  []int{7, 11, 13},
				Money:    123.45,
				Position: 1 + 2i,
				Created:  timestamp,
			},
		},
		{
			given: url.Values{
				"name":     {"Mateus"},
				"age":      {"30"},
				"balance":  {"65"},
				"hidden":   {"T"},
				"tag":      {"admin", "ops"},
				"position": {"1-2i"},
				"created":  {timestamp.Format(time.RFC3339)},
			},
			expected: Input{
				Name:     "Mateus",
				Age:      30,
				Balance:  65,
				Hidden:   true,
				Tags:     []string{"admin", "ops"},
				Position: 1 - 2i,
				Created:  timestamp,
			},
		},
	}

	for _, test := range tests {
		var v Input
		if err := Parse(test.given, &v); err != nil {
			t.Error(err)
			continue
		}

		if !reflect.DeepEqual(v, test.expected) {
			t.Errorf("Unexpected result:\n\tgot:  %v\n\twant: %v", v, test.expected)
		}
	}
}
