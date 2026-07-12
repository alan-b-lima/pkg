package problem

import (
	"maps"
	"testing"
)

func Test_Details(t *testing.T) {
	type Test struct {
		name string
		got  []map[string]any
		want map[string]any
	}

	tests := []Test{
		{
			name: "empty",
			got:  []map[string]any{},
			want: map[string]any{},
		},
		{
			name: "inserting",
			got: []map[string]any{
				{"a": "Hello, World"},
				{"b": "Hello, World"},
			},
			want: map[string]any{"a": "Hello, World", "b": "Hello, World"},
		},
		{
			name: "replacing",
			got: []map[string]any{
				{"a": "Hello, World", "b": "Hello, World"},
				{"a": "Goodbye, World"},
			},
			want: map[string]any{"a": "Goodbye, World", "b": "Hello, World"},
		},
		{
			name: "deleting",
			got: []map[string]any{
				{"a": "Hello, World", "b": "Hello, World"},
				{"b": nil},
			},
			want: map[string]any{"a": "Hello, World"},
		},
		{
			name: "all",
			got: []map[string]any{
				{"a": "Hello, World", "b": "Hello, World"},
				{"c": "Hello, World"},
				{"b": "Goodbye, World"},
				{"a": nil},
			},
			want: map[string]any{"b": "Goodbye, World", "c": "Hello, World"},
		},
	}

	for _, test := range tests {
		var imp ImpError
		for _, d := range test.got {
			imp = imp.Details(d)
		}

		if !maps.Equal(imp.details, test.want) {
			t.Errorf("%s: different result:\n\tgot:  %#v\n\twant: %#v", test.name, imp.details, test.want)
		}
	}
}
