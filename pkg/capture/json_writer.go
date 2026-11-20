package capture

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
)

// JSONWriter streams records as one JSON array without retaining the capture in
// memory. The writer does not close the supplied io.Writer; callers retain
// ownership of it.
//
// A capture is finalized by Close. Before Close, the output contains the
// streamed array prefix and any records written so far. Flush makes that
// partial output available to readers while the capture is still running.
type JSONWriter struct {
	writer *bufio.Writer
	output io.Writer
	first  bool
	closed bool
	err    error
}

// NewJSONWriter creates a streaming JSON-array writer around w.
func NewJSONWriter(w io.Writer) *JSONWriter {
	writer := bufio.NewWriter(w)
	result := &JSONWriter{writer: writer, output: w, first: true}
	if _, err := writer.WriteString("[\n"); err != nil {
		result.err = err
	}
	return result
}

// Write appends one JSON record and returns the number of encoded record bytes
// written, including its array separator but excluding the array delimiters.
func (w *JSONWriter) Write(record any) (int, error) {
	if w.closed {
		return 0, errors.New("write to closed JSON writer")
	}
	if w.err != nil {
		return 0, w.err
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		w.err = err
		return 0, err
	}
	return w.writeEncoded(encoded)
}

// WriteJSON appends one already-serialized JSON record. It is intended for
// pipeline records that arrive as JSON bytes; unlike Write([]byte), it does
// not encode the bytes as a base64 JSON string.
func (w *JSONWriter) WriteJSON(record []byte) (int, error) {
	if w.closed {
		return 0, errors.New("write to closed JSON writer")
	}
	if w.err != nil {
		return 0, w.err
	}
	if !json.Valid(record) {
		w.err = errors.New("invalid JSON record")
		return 0, w.err
	}
	return w.writeEncoded(record)
}

func (w *JSONWriter) writeEncoded(encoded []byte) (int, error) {
	if w.closed {
		return 0, errors.New("write to closed JSON writer")
	}
	if w.err != nil {
		return 0, w.err
	}

	var err error
	n := 0
	if !w.first {
		if _, err = w.writer.WriteString(",\n"); err != nil {
			w.err = err
			return n, err
		}
		n += 2
	}
	if _, err = w.writer.Write(encoded); err != nil {
		w.err = err
		return n, err
	}
	n += len(encoded)
	w.first = false
	return n, nil
}

// Flush publishes all records written so far while keeping the array open.
func (w *JSONWriter) Flush() error {
	if w.err != nil {
		return w.err
	}
	if err := w.writer.Flush(); err != nil {
		w.err = err
		return err
	}
	if flusher, ok := w.output.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); err != nil {
			w.err = err
			return err
		}
	}
	return nil
}

// Close finalizes the JSON array and flushes it. It is safe to call Close more
// than once; subsequent calls return the result of the first close.
func (w *JSONWriter) Close() error {
	if w.closed {
		return w.err
	}
	w.closed = true
	if w.err == nil {
		if _, w.err = w.writer.WriteString("\n]\n"); w.err == nil {
			w.err = w.Flush()
		}
	}
	return w.err
}
