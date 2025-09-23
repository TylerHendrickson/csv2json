package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"
)

// JSON delimiters
const (
	jsonDelimArrayOpen  byte = '['
	jsonDelimArrayClose byte = ']'
	jsonDelimArraySep   byte = ','
	jsonDelimNewline    byte = '\n'
)

// Common JSON sequences
var (
	jsonEmptyArray = []byte{'[', ']'}
	jsonNullRecord = []byte("null")
)

// ErrJSONRecordWriterClosed is returned by jsonRecordWriter methods after Close has been called.
var ErrJSONRecordWriterClosed = errors.New("write after close")

type jsonRecordWriter struct {
	w                  *bufio.Writer // usually to stdout
	asArray            bool          // newline-delimited mode when false
	mux                sync.Mutex
	firstRecordWritten bool
	closed             bool
}

// NewJSONRecordWriter returns a new *jsonRecordWriter with an underlying buffer that writes JSON records to w.
// If asArray is true, records are written as a single JSON array using streaming delimiters ([, , , ]).
// If asArray is false, each record is written on its own line as line-delimited JSON.
// The caller must call Close() when finished to ensure the output is properly finalized and flushed.
func NewJSONRecordWriter(dst io.Writer, asArray bool, size int) *jsonRecordWriter {
	var bw *bufio.Writer
	if size > 0 {
		bw = bufio.NewWriterSize(dst, size)
	} else {
		bw = bufio.NewWriter(dst)
	}
	return &jsonRecordWriter{w: bw, asArray: asArray}
}

func NewJSONRecordWriterSize(w io.Writer, asArray bool, size int) *jsonRecordWriter {
	return &jsonRecordWriter{w: bufio.NewWriterSize(w, size), asArray: asArray}
}

// WriteRecord converts rec to JSON and writes the resulting bytes data to the buffer.
func (jrw *jsonRecordWriter) WriteRecord(rec map[string]string) (n int, err error) {
	data, err := json.Marshal(rec)
	if err != nil {
		return 0, err
	}
	return jrw.Write(data)
}

// WriteNull writes a JSON null reocrd to the buffer.
func (jrw *jsonRecordWriter) WriteNull() (n int, err error) {
	return jrw.Write(jsonNullRecord)
}

// Write writes data to the buffer along with framing according to array/newline-delimited mode.
// It returns the number of bytes written from data (excluding framing bytes for compatibility
// with io.Writer). which may be less than len(data) if an error occurs.
// Write returns ErrJSONRecordWriterClosed if called after Close.
func (jrw *jsonRecordWriter) Write(data []byte) (int, error) {
	jrw.mux.Lock()
	defer jrw.mux.Unlock()

	if jrw.closed {
		return 0, ErrJSONRecordWriterClosed
	}

	if len(data) == 0 {
		// Nothing to write
		return 0, nil
	}

	if jrw.asArray {
		if !jrw.firstRecordWritten {
			if err := jrw.w.WriteByte(jsonDelimArrayOpen); err != nil {
				return 0, err
			}
			jrw.firstRecordWritten = true
		} else {
			if err := jrw.w.WriteByte(jsonDelimArraySep); err != nil {
				return 0, err
			}
		}
		return jrw.w.Write(data)
	}

	// Line-delimited mode
	n, err := jrw.w.Write(data)
	if err != nil {
		return n, err
	}
	if err := jrw.w.WriteByte(jsonDelimNewline); err != nil {
		return n, err
	}
	return n, err
}

// Close finalizes output and flushes the buffer. In array mode, it emits an empty array if
// no records have been written, or else closes the array.
// It is safe to call this method more than once; subsequent calls return immediately.
func (jrw *jsonRecordWriter) Close() error {
	jrw.mux.Lock()
	defer jrw.mux.Unlock()
	if jrw.closed {
		return nil
	}
	jrw.closed = true

	if jrw.asArray {
		if !jrw.firstRecordWritten {
			if _, err := jrw.w.Write(jsonEmptyArray); err != nil {
				return err
			}
		} else {
			if err := jrw.w.WriteByte(jsonDelimArrayClose); err != nil {
				return err
			}
		}
	}

	return jrw.w.Flush()
}

// Flush flushes the underlying buffer.
func (jrw *jsonRecordWriter) Flush() error {
	jrw.mux.Lock()
	defer jrw.mux.Unlock()
	return jrw.w.Flush()
}

// startPeriodicFlush starts a goroutine that flushes w every d until stop is called.
// If d <= 0, no flushing will occur and stop is a no-op.
// It is safe to call stop() multiple times.
func startPeriodicFlush(w interface{ Flush() error }, d time.Duration) (stop func()) {
	if d <= 0 {
		return func() {}
	}

	done := make(chan struct{})
	go func() {
		// If already stopped, skip creating the ticker.
		select {
		case <-done:
			return
		default:
		}

		t := time.NewTicker(d)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				_ = w.Flush()
			case <-done:
				return
			}
		}
	}()

	var once sync.Once
	return func() { once.Do(func() { close(done) }) }
}
