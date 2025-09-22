package main

import (
	"encoding"
	"fmt"
	"strings"
)

// OnErrorAction represent a behavior for the application to follow
// when an error results from reading CSV input.
type OnErrorAction int

// Compile-time assertion
var _ encoding.TextUnmarshaler = (*OnErrorAction)(nil)

const (
	allow OnErrorAction = iota // Output a transformed record despite any error
	null                       // Emit a null record rather than transforming the parsed content
	skip                       // Do not emit any output for the record
	abort                      // Terminate the application due to the error
)

// parseOnErrorAction converts an OnErrorAction string into an OnErrorAction value.
// Returns an error if s does not match known values.
func parseOnErrorAction(s string) (action OnErrorAction, err error) {
	lower := strings.ToLower(s)
	for _, action := range []OnErrorAction{abort, allow, skip, null} {
		if action.String() == lower {
			return action, nil
		}
	}
	err = fmt.Errorf("unknown error action: %s", s)
	return
}

func (d *OnErrorAction) UnmarshalText(text []byte) (err error) {
	*d, err = parseOnErrorAction(string(text))
	return
}

// toStrings creates a []string from any number of stringers.
// It is useful for succinctly creating slices that can be passed to strings.Join().
func toStrings(in ...fmt.Stringer) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = v.String()
	}
	return out
}

// joinStringers is a convenience wrapper for using toStrings() with strings.Join().
func joinStringers(sep string, elems ...fmt.Stringer) string {
	return strings.Join(toStrings(elems...), sep)
}
