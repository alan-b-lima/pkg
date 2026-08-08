package sse_test

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"net/http"
	"slices"
	"testing"

	. "github.com/alan-b-lima/pkg/sse"
)

type mockResponseWriter struct {
	buf []byte
}

func (m *mockResponseWriter) Header() http.Header { return http.Header{} }
func (m *mockResponseWriter) WriteHeader(int)     {}
func (m *mockResponseWriter) Flush()              {}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	m.buf = append(m.buf, b...)
	return len(b), nil
}

func Fuzz_Write(f *testing.F) {
	f.Add([]byte("hello\nworld"))
	f.Add([]byte("hello"))
	f.Add([]byte("hello\nworld\r\neverybody!"))
	f.Add([]byte("hello\n\rwo\rrld\r\neverybody!"))

	f.Fuzz(func(t *testing.T, b []byte) {
		var m mockResponseWriter
		sse, err := New(&m)
		if err != nil {
			t.Errorf("New should not error: %v", err)
		}

		if _, err := sse.Write(b); err != nil {
			t.Errorf("Write should not error: %v", err)
		}
	})
}

func Test_Write(t *testing.T) {
	type Template struct {
		write   []byte
		expect  []byte
		samples int
	}

	type Test struct {
		writes [][]byte
		expect []byte
	}

	rand := rand.New(rand.NewPCG(0, 2))

	templates := []Template{
		{
			write:   []byte("hello"),
			expect:  lines("data: hello"),
			samples: 30,
		},
		{
			write:   []byte("hello\nworld"),
			expect:  lines("data: hello", "data: world"),
			samples: 30,
		},
		{
			write:   []byte("hello\nworld\r\neverybody!"),
			expect:  lines("data: hello", "data: world", "data: everybody!"),
			samples: 30,
		},
		{
			write:   []byte("hello\n\rwo\rrld\r\neverybody!"),
			expect:  lines("data: hello", "data: world", "data: everybody!"),
			samples: 30,
		},
		{
			write:   []byte("hello\n\rwo\rrld\n\reverybody!"),
			expect:  lines("data: hello", "data: world", "data: everybody!"),
			samples: 30,
		},
	}

	var tests []Test
	for _, template := range templates {
		for range template.samples {
			tests = append(tests, Test{
				writes: fragment(template.write, 1+rand.IntN(len(template.write)), rand),
				expect: template.expect,
			})
		}
	}

	for _, test := range tests {
		var m mockResponseWriter
		sse, err := New(&m)
		if err != nil {
			t.Errorf("New should not error: %v", err)
		}

		for _, write := range test.writes {
			_, err := sse.Write(write)
			if err != nil {
				t.Errorf("Write should not error: %v", err)
			}
		}

		_, err = sse.Dispatch()
		if err != nil {
			t.Errorf("Dispatch should not error: %v", err)
		}

		if !bytes.Equal(m.buf, test.expect) {
			t.Errorf(
				"unexpected output:\n\twrites:   %+q\n\tobtained: %+q\n\texpected: %+q",
				test.writes, m.buf, test.expect,
			)
		}
	}
}

func Test_TypeAndID(t *testing.T) {
	type Test struct {
		got  string
		want string
	}

	tests := []Test{
		{got: "", want: ""},
		{got: "a", want: "a"},
		{got: "a\nHello", want: "a"},
		{got: "a  ", want: "a  "},
		{got: "a \000 Something", want: "a "},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Type#%d", i), func(t *testing.T) {
			var sse Dispatcher
			sse.SetType(test.got)

			if sse.Type() != test.want {
				t.Errorf("unexpected output:\n\tgot:  %+q\n\twant: %+q", test.got, test.want)
			}
		})

		t.Run(fmt.Sprintf("ID#%d", i), func(t *testing.T) {
			var sse Dispatcher
			sse.SetID(test.got)

			if sse.ID() != test.want {
				t.Errorf("unexpected output:\n\tgot:  %+q\n\twant: %+q", test.got, test.want)
			}
		})
	}
}

func Test_Sequence(t *testing.T) {
	type (
		ID       string
		Type     string
		Data     []byte
		Dispatch struct{}
	)

	type Test struct {
		Actions []any
		Result  []byte
	}

	tests := []Test{
		{
			Actions: []any{
				ID("234"),
				Type("del"),
				Data("Hello,\nWorld"),
				Dispatch{},
				Type("add"),
				Data("Goodbye,\nWorld"),
				Dispatch{},
			},
			Result: lines(
				"id: 234",
				"type: del",
				"data: Hello,",
				"data: World",
				"",
				"type: add",
				"data: Goodbye,",
				"data: World",
			),
		},
		{
			Actions: []any{
				Type("del"),
				ID("234"),
				ID("235"),
				Data("Hello,\nWorld"),
				ID("236"),
				Dispatch{},
				Data("Goodbye,\nWorld"),
				Type("add"),
				ID("237"),
				Dispatch{},
			},
			Result: lines(
				"id: 236",
				"type: del",
				"data: Hello,",
				"data: World",
				"",
				"id: 237",
				"type: add",
				"data: Goodbye,",
				"data: World",
			),
		},
	}

	for _, test := range tests {
		var m mockResponseWriter
		sse, _ := New(&m)

		for _, action := range test.Actions {
			switch action := action.(type) {
			case ID:
				sse.SetID(string(action))

			case Type:
				sse.SetType(string(action))

			case Data:
				sse.Write(action)

			case Dispatch:
				sse.Dispatch()
			}
		}

		if bytes.Equal(m.buf, test.Result) {
			t.Errorf("unexpected output:\n\tgot:  %+q\n\twant: %+q", m.buf, test.Result)
		}
	}
}

func fragment(expr []byte, pieces int, rand *rand.Rand) [][]byte {
	if len(expr) < pieces {
		pieces = len(expr)
	}

	cuts := make([]int, 1, pieces+1)
	for range pieces - 1 {
		cuts = append(cuts, rand.IntN(len(expr)))
	}
	cuts = append(cuts, len(expr))

	slices.Sort(cuts)

	fragments := make([][]byte, 0, pieces)
	for i := range pieces {
		lo, hi := cuts[i], cuts[i+1]
		fragments = append(fragments, expr[lo:hi])
	}

	return fragments
}

func lines(ss ...string) []byte {
	var b bytes.Buffer
	for _, s := range ss {
		b.WriteString(s)
		b.WriteString("\r\n")
	}

	b.WriteString("\r\n")
	return b.Bytes()
}
