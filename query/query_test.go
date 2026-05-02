package query_test

import (
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
	}

	tests := []Test{
		{Name: "nil-pointer", Input: nil},
		{Name: "non-pointer", Input: struct{}{}},
		{Name: "pointer to non-struct", Input: new(int)},
		{Name: "nil pointer to struct", Input: (*struct{})(nil)},
	}

	for _, test := range tests {
		err := Unmarshal(url.Values{"name": {"alice"}}, test.Input)
		if err == nil {
			t.Errorf("query test %+q shouldn't have succeded", test.Name)
		}
	}
}

func TestParsing(t *testing.T) {
	type Input struct {
		Name     string     `query:"name"`
		Age      int        `query:"age"`
		Height   float64    `query:"height"`
		Balance  uint32     `query:"balance"`
		Hidden   bool       `query:"hidden"`
		Tags     []string   `query:"tag"`
		Numbers  []int      `query:"num"`
		Position complex128 `query:"position"`
		Created  time.Time  `query:"created"`
	}

	type Test struct {
		Given url.Values
		Want  Input
	}

	timestamp := time.Date(2026, time.March, 14, 15, 92, 65, 0, time.UTC)

	tests := []Test{
		{
			Given: url.Values{
				"name":     {"Luan"},
				"age":      {"23"},
				"height":   {"1.73"},
				"balance":  {"900"},
				"hidden":   {"false"},
				"tag":      {"admin", "ops"},
				"num":      {"7", "11", "13"},
				"position": {"1+2i"},
				"created":  {timestamp.Format(time.RFC3339Nano)},
			},
			Want: Input{
				Name:     "Luan",
				Age:      23,
				Height:   1.73,
				Balance:  900,
				Hidden:   false,
				Tags:     []string{"admin", "ops"},
				Numbers:  []int{7, 11, 13},
				Position: 1 + 2i,
				Created:  timestamp,
			},
		},
		{
			Given: url.Values{
				"name":     {"Mateus"},
				"age":      {"30"},
				"balance":  {"65"},
				"hidden":   {"T"},
				"tag":      {"admin", "ops"},
				"position": {"5-2i"},
				"created":  {timestamp.Format(time.RFC3339)},
			},
			Want: Input{
				Name:     "Mateus",
				Age:      30,
				Balance:  65,
				Hidden:   true,
				Tags:     []string{"admin", "ops"},
				Position: 5 - 2i,
				Created:  timestamp,
			},
		},
	}

	for _, test := range tests {
		var v Input
		if err := Unmarshal(test.Given, &v); err != nil {
			t.Error(err)
			continue
		}

		if !reflect.DeepEqual(v, test.Want) {
			t.Errorf("Unexpected result:\n\tgot:  %v\n\twant: %v", v, test.Want)
		}
	}
}
