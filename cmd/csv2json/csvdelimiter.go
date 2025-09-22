package main

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// CSVDelimiter represents a character used to delimit CSV field separators (e.g. commas)
// or commented lines.
type CSVDelimiter rune

// Compile-time assertion
var _ encoding.TextUnmarshaler = (*CSVDelimiter)(nil)

func (d *CSVDelimiter) UnmarshalText(text []byte) error {
	var delim CSVDelimiter

	if bytes.EqualFold(text, []byte("tab")) {
		delim = '\t'
	} else if count := utf8.RuneCount(text); count != 1 {
		return fmt.Errorf("must be exactly 1 unicode character (got %d)", count)
	} else if r, n := utf8.DecodeRune(text); n > 0 {
		delim = CSVDelimiter(r)
	}
	*d = delim
	return nil
}

// Validate returns an error if d is not a valid CSV delimiter.
func (d *CSVDelimiter) Validate() error {
	if r := d.Rune(); r == '\n' || r == '\r' || r == 0xFFFD {
		// Error string comes from csv.Reader docs
		return fmt.Errorf("must not be %q, %q, or the Unicode replacement character %q",
			'\n', '\r', 0xFFFD)
	}
	return nil
}

func (d *CSVDelimiter) Rune() rune {
	return rune(*d)
}

func (d *CSVDelimiter) String() string {
	return string(d.Rune())
}

func (d *CSVDelimiter) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}
