package capture

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONWriterEmpty(t *testing.T) {
	var output bytes.Buffer
	writer := NewJSONWriter(&output)
	require.NoError(t, writer.Close())
	require.JSONEq(t, "[]", output.String())
}

func TestJSONWriterOneRecord(t *testing.T) {
	var output bytes.Buffer
	writer := NewJSONWriter(&output)
	_, err := writer.Write(map[string]int{"value": 1})
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.JSONEq(t, `[{"value":1}]`, output.String())
}

func TestJSONWriterRawJSONRecord(t *testing.T) {
	var output bytes.Buffer
	writer := NewJSONWriter(&output)
	_, err := writer.WriteJSON([]byte(`{"value":1}`))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.JSONEq(t, `[{"value":1}]`, output.String())
}

func TestJSONWriterRecords(t *testing.T) {
	var output bytes.Buffer
	writer := NewJSONWriter(&output)
	_, err := writer.Write(map[string]any{"value": 1})
	require.NoError(t, err)
	_, err = writer.Write(map[string]any{"value": 2})
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.JSONEq(t, `[{"value":1},{"value":2}]`, output.String())
	require.True(t, json.Valid(output.Bytes()))
}

func TestJSONWriterFlushesPartialOutput(t *testing.T) {
	var output bytes.Buffer
	writer := NewJSONWriter(&output)
	_, err := writer.Write(map[string]string{"value": "partial"})
	require.NoError(t, err)
	require.NoError(t, writer.Flush())
	require.Equal(t, "[\n{\"value\":\"partial\"}", output.String())
	require.False(t, json.Valid(output.Bytes()))
	require.NoError(t, writer.Close())
	require.True(t, json.Valid(output.Bytes()))
}

func TestJSONWriterCloseFlushesAndDoesNotCloseOutput(t *testing.T) {
	output := &flushTrackingWriter{}
	writer := NewJSONWriter(output)
	_, err := writer.Write(map[string]bool{"ok": true})
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.Equal(t, 1, output.flushes)
	require.NoError(t, writer.Close())
	_, err = writer.Write(nil)
	require.Error(t, err)
	require.NotNil(t, output.writer)
}

type flushTrackingWriter struct {
	writer  bytes.Buffer
	flushes int
}

func (w *flushTrackingWriter) Write(p []byte) (int, error) { return w.writer.Write(p) }
func (w *flushTrackingWriter) Flush() error                { w.flushes++; return nil }

func TestJSONWriterPropagatesMarshalError(t *testing.T) {
	var output bytes.Buffer
	writer := NewJSONWriter(&output)
	_, err := writer.Write(func() {})
	require.Error(t, err)
	require.Error(t, writer.Close())
}

func TestJSONWriterPropagatesWriterError(t *testing.T) {
	writer := NewJSONWriter(errorWriter{})
	_, err := writer.Write(map[string]bool{"ok": true})
	require.NoError(t, err)
	require.Error(t, writer.Close())
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
