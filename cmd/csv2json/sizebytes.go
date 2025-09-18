package main

import (
	"strings"

	"github.com/docker/go-units"
)

type SizeBytes int

func (s *SizeBytes) UnmarshalText(b []byte) error {
	str := strings.TrimSpace(string(b))

	// Prefer 1024-based units (e.g. "4k" == 4096)
	n, err := units.RAMInBytes(str)
	*s = SizeBytes(n)
	return err
}
