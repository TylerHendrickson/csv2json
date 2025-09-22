package main

import (
	"encoding"
	"strings"

	"github.com/docker/go-units"
)

// SizeBytes represents an amount of data in bytes.
type SizeBytes int

// Compile-time assertion
var _ encoding.TextUnmarshaler = (*SizeBytes)(nil)

func (s *SizeBytes) UnmarshalText(b []byte) error {
	str := strings.TrimSpace(string(b))

	// Prefer 1024-based units (e.g. "4k" == 4096)
	n, err := units.RAMInBytes(str)
	*s = SizeBytes(n)
	return err
}
