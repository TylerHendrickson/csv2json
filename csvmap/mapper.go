package csvmap

import (
	"encoding/csv"
	"errors"

	"github.com/TylerHendrickson/csv2json/assoc"
)

// CSVMapReader is an alias for assoc.MapReader in the context of working with CSVs.
type CSVMapReader = assoc.MapReader[string, string]

// RowResult provides the results of reading and parsing a CSV row.
type RowResult struct {
	// A single CSV row's worth of data, keyed by field names.
	Record
	// Error encountered when reading or mapping the Record.
	Err error
	// Starting line number the reader parsed when reading Record or encountering Err.
	// Line is always 0 when Err == io.EOF.
	Line int
}

type rowSource interface {
	// From assoc
	Read() ([]string, error)
	// from encoding/csv.Reader
	FieldPos(field int) (line, col int)
}

// Mapper reads CSV rows returns each as a fieldname-keyed record.
// Construct with NewMapper and iterate by calling Next, which yields a RowResult containing
// the mapped record, any error, and line number that produced the record and/or error
// (line number is always zero when error is io.EOF).
//
// Mapper reads sequentially from a single *csv.Reader. Calls to its methods must not be made concurrently
// from multiple goroutines. If you need concurrency, perform reads in one goroutine and distribute RowResults
// via channels.
type Mapper struct {
	m CSVMapReader
	r rowSource
}

// New returns a new *Mapper that maps values from r to fields.
func New(r *csv.Reader, fields []string) *Mapper {
	return &Mapper{
		m: assoc.NewMapReader(r, fields),
		r: r,
	}
}

// NewFromRowSource is basically equivalent to New(), but slightly more flexible
// as it expects a minimal rowSource implemenation rather than requiring a concrete *csv.Reader.
func NewFromRowSource(rs rowSource, fields []string) *Mapper {
	return &Mapper{
		m: assoc.NewMapReader(rs, fields),
		r: rs,
	}
}

// Read implements CSVMapReader.Read.
func (m *Mapper) Read() (map[string]string, error) {
	return m.m.Read()
}

// Next returns the next rows result (record, error, line).
func (m *Mapper) Next() RowResult {
	res := RowResult{}
	res.Record, res.Err = m.Read()

	var pe *csv.ParseError
	switch {
	case errors.As(res.Err, &pe) && pe.StartLine > 0:
		res.Line = pe.StartLine
	case res.Record != nil:
		if line, _ := m.r.FieldPos(0); line > 0 {
			res.Line = line
		}
	default:
		// Likely EOF
	}

	return res
}
