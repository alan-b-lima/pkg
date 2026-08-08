package sse

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
)

// Dispatcher is a struct that represents a Server-Sent Event connection.
// It provides methods to safely write data to the connection and dispatch
// events to the client. Implemented accounding to the [SSE Specification].
//
// Dispatcher is NOT safe for concurrent use.
//
// [SSE Specification]: https://html.spec.whatwg.org/multipage/server-sent-events.html
type Dispatcher struct {
	writer  io.Writer
	flusher flusher

	id  string
	typ string
	buf bytes.Buffer

	idsent  bool
	typsent bool
	bufsent bool
}

var ErrUnsupported = errors.New("sse: streaming unsupported")

// New creates a new Dispatcher and attaches it to the provided
// [http.ResponseWriter].
//
// New takes ownership over the http.ResponseWriter and the caller should not
// use it after this call.
//
// It returns an error if the provided http.ResponseWriter does not support
// the necessary interfaces for streaming.
func New(w http.ResponseWriter) (*Dispatcher, error) {
	var e Dispatcher
	if err := e.Attach(w); err != nil {
		return nil, err
	}

	return &e, nil
}

// NewBuffer creates a new Dispatcher and attaches it to the provided
// [http.ResponseWriter].
//
// NewBuffer takes ownership over the http.ResponseWriter and the caller should
// not use it after this call.
//
// The given byte slice is used as the backing for the buffered data, the
// Dispatcher takes complete ownership of the slice.
//
// It returns an error if the provided http.ResponseWriter does not support
// the necessary interfaces for streaming.
func NewBuffer(w http.ResponseWriter, b []byte) (*Dispatcher, error) {
	e := Dispatcher{
		buf: *bytes.NewBuffer(b[:0]),
	}

	if err := e.Attach(w); err != nil {
		return nil, err
	}

	return &e, nil
}

// Attach attaches the Dispatcher to the provided [http.ResponseWriter]. Attach
// takes ownership over the http.ResponseWriter and the caller should not use
// it after this call.
//
// It returns an error if the provided http.ResponseWriter does not support
// the necessary interfaces for streaming.
//
// Note that the object used for writting (dispatching) is always the one given
// to this function, while the one used for flushing is the first one found in
// the Unwrap() http.ResponseWriter chain.
func (e *Dispatcher) Attach(w http.ResponseWriter) error {
	f, ok := toFlusher(w)
	if !ok {
		return ErrUnsupported
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Add("Content-Type", "text/event-stream")
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Add("Connection", "keep-alive")

	e.writer = w
	e.flusher = f

	e.id = w.Header().Get("Last-Event-Id")
	e.typ = ""

	e.idsent = true
	e.typsent = true
	e.bufsent = true

	return nil
}

// Detach detaches the Dispatcher from its current connection.
//
// Detaching will be reset the Dispatcher, leaving it with no ID, type, or
// buffered data. The Dispatcher will solely keep the buffer capacity.
//
// Detaching does not close the underlying connection, but it will prevent any
// further dispatches from affecting the connection.
//
// Detach is not required to be run before attaching to a new connection, but
// may be used to allow the underlying response writer to be claimed by the
// garbage collector.
func (e *Dispatcher) Detach() {
	buf := e.buf
	*e = Dispatcher{}

	buf.Reset()
	e.buf = buf
}

// Type returns the current event type of the Dispatcher, or an empty string
// if no type has been set.
func (e *Dispatcher) Type() string { return e.typ }

// ID returns the current event ID of the Dispatcher, or an empty string if
// no ID has been set.
func (e *Dispatcher) ID() string { return e.id }

// SetType sets the event type of the Dispatcher.
//
// The event type will be sent to the client on the next dispatch if it has
// been set, is not empty and not already sent.
//
// If the provided type contains any of null terminator (NUL U+0000), line feed
// (LF U+000A) or carriage return (CR U+000D), only the substring up to the
// first these, exclusive, will be used, and the rest will be ignored.
func (e *Dispatcher) SetType(typ string) {
	index := strings.IndexAny(typ, "\000\r\n")
	if index == -1 {
		index = len(typ)
	}

	e.typsent = false
	e.typ = typ[:index]
}

// SetID sets the event ID of the Dispatcher.
//
// The event ID will be sent to the client on the next dispatch if it has been
// set and not already sent.
//
// If the provided type contains any of null terminator (NUL U+0000), line feed
// (LF U+000A) or carriage return (CR U+000D), only the substring up to the
// first these, exclusive, will be used, and the rest will be ignored.
func (e *Dispatcher) SetID(id string) {
	index := strings.IndexAny(id, "\000\r\n")
	if index == -1 {
		index = len(id)
	}

	e.idsent = false
	e.id = id[:index]
}

var (
	id      = []byte("id: ")
	event   = []byte("event: ")
	data    = []byte("data: ")
	comment = []byte(":")

	idping   = []byte("id\r\n")
	dataping = []byte("data\r\n\r\n")

	crlf     = []byte("\r\n")
	crlfcrlf = []byte("\r\n\r\n")
)

// Write writes the provided byte slice to the Dispatcher's buffer, properly
// formatting it according to the Server-Sent Events specification.
//
// Server-Sent Events SHOULD NOT be used to send arbitrary binary data, the
// stream is always interpreted as UTF-8; and carriage returns U+000D are
// impossible to send, therefore ignored here to avoid problems down the line.
//
// The provided byte slice will be automatically split into multiple data lines
// if it contains newline characters.
//
// err is always nil. If the buffer becomes too large, Write will panic with
// [bytes.ErrTooLarge].
//
// Write does not automatically dispatch the event to the client, users have to
// call [Dispatcher.Dispatch] to send the buffered data.
func (e *Dispatcher) Write(b []byte) (int, error) {
	e.bufsent = false
	if len(b) == 0 {
		return 0, nil
	}

	e.buf.Grow(len(b))
	orglen := e.buf.Len()

	if e.buf.Len() == 0 {
		e.buf.Write(data)
	}

	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '\r':
			e.buf.Write(b[:i])
			b = b[i+1:]
			i = -1

		case '\n':
			e.buf.Write(b[:i])
			e.buf.Write(crlf)
			e.buf.Write(data)

			b = b[i+1:]
			i = -1
		}
	}

	e.buf.Write(b)
	return orglen - e.buf.Len(), nil
}

// Dispatch sends the buffered event data to the client, properly formatting it
// according to the Server-Sent Events specification.
//
// Dispatch will also send the event's ID and type if they have been set and
// not already sent.
//
// After dispatching, the Dispatcher's buffer and type will be cleared, but the
// ID will remain set.
func (e *Dispatcher) Dispatch() (int, error) {
	n, err := e.dispatch()
	if err != nil {
		if n > 0 {
			e.flusher.Flush()
		}
		return n, err
	}

	if n > 0 {
		return n, e.flusher.Flush()
	}
	return n, nil
}

func (e *Dispatcher) dispatch() (int, error) {
	var ntotal int

	if !e.idsent {
		var err error
		if e.id != "" {
			err = e.writeBufs(&ntotal, id, []byte(e.id), crlf)
		} else {
			err = e.writeBufs(&ntotal, idping)
		}

		if err != nil {
			return ntotal, err
		}

		e.idsent = true
	}

	if !e.typsent && e.typ != "" {
		err := e.writeBufs(&ntotal, event, []byte(e.typ), crlf)
		if err != nil {
			return ntotal, err
		}

		e.typ = ""
		e.typsent = true
	}

	if !e.bufsent {
		if e.buf.Len() == 0 {
			err := e.writeBufs(&ntotal, dataping)
			return ntotal, err
		}

		err := e.writeBufs(&ntotal, e.buf.Bytes(), crlfcrlf)
		if err != nil {
			return ntotal, err
		}

		e.buf.Reset()
		e.bufsent = true
	}

	return ntotal, nil
}

// Comment writes the provided byte slice as a comment to the Dispatcher's
// connection, properly formatting it according to the Server-Sent Events
// specification.
//
// Comments have no special meaning and are ignored by the client, but they can
// be used to keep the connection alive. You may use Comment(nil) to send a
// ping comment, which is a comment with no content.
//
// See [Dispatcher.Write] for the allowed (recommended) data to be transmitted.
func (e *Dispatcher) Comment(b []byte) (int, error) {
	defer e.flusher.Flush()
	var ntotal int

	err := e.writeBufs(&ntotal, comment)
	if err != nil {
		return ntotal, err
	}

	if len(b) == 0 {
		return ntotal, nil
	}

	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '\r':
			err := e.writeBufs(&ntotal, b[:i])
			if err != nil {
				return ntotal, err
			}

			b = b[i+1:]
			i = -1

		case '\n':
			err := e.writeBufs(&ntotal, b[:i], crlf, comment)
			if err != nil {
				return ntotal, err
			}

			b = b[i+1:]
			i = -1
		}
	}

	err = e.writeBufs(&ntotal, b)
	return ntotal, err
}

type flusher interface {
	Flush() error
}

type flusherErr struct {
	http.Flusher
}

func (f *flusherErr) Flush() error {
	f.Flusher.Flush()
	return nil
}

func toFlusher(rw http.ResponseWriter) (flusher, bool) {
	for {
		switch t := rw.(type) {
		case http.Flusher:
			return &flusherErr{Flusher: t}, true

		case flusher:
			return t, true

		case interface{ Unwrap() http.ResponseWriter }:
			rw = t.Unwrap()

		default:
			return nil, false
		}
	}
}

func (e *Dispatcher) writeBufs(n *int, p ...[]byte) (err error) {
	for _, b := range p {
		m, err := e.writer.Write(b)
		*n += m

		if err != nil {
			return err
		}
	}

	return nil
}
