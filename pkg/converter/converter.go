package converter

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"io"
	basicLog "log"
)

// record values are a single row's worth of data, keyed by column names
type record map[string]string

type Options struct {
	ColNames   []string
	CsvInput   io.Reader
	JsonOutput io.Writer
	SkipErrors bool
	Logger     *log.Logger
}

func Execute(o Options) error {
	//return csv2Json(o)
	return Instrument(&o)
}

func Instrument(o *Options) error {
	reader, err := getCsvReader(o.CsvInput)
	if err != nil {
		return err
	}

	if len(o.ColNames) == 0 {
		columnNames, err := parseColumnNames(reader)
		if err != nil {
			return err
		}
		o.ColNames = columnNames
	}
	reader.FieldsPerRecord = len(o.ColNames)

	records, err := buildRecords(reader, o)
	if err != nil {
		return err
	}

	return json.NewEncoder(o.JsonOutput).Encode(&records)
}

func parseColumnNames(reader *csv.Reader) ([]string, error) {
	firstRow, err := reader.Read()
	if err != nil && err != io.EOF {
		if err != io.EOF {
			return nil, err
		}
	}

	return firstRow, nil
}

func buildRecords(reader *csv.Reader, o *Options) (records []record, err error) {
	skipped := 0
	records = make([]record, 0)
	for {
		rowFields, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				err = nil
			} else if o.SkipErrors {
				skipped++
				level.Error(*o.Logger).Log("message", "Skipped parsing error", "error", err)
				continue
			}
			break
		} else {
			records = append(records, fieldsToRecord(&o.ColNames, &rowFields))
		}
	}

	if skipped > 0 {
		level.Info(*o.Logger).Log("message", "Skipped lines (rows) due to parsing errors", "count", skipped)
	}

	return
}

func csv2Json(o Options) error {
	reader, err := getCsvReader(o.CsvInput)
	if err != nil {
		return err
	}

	colNames := o.ColNames
	if len(colNames) == 0 {
		// Read the first line to get column names
		if firstRow, err := reader.Read(); err != nil {
			if err != io.EOF {
				return err
			}
		} else {
			colNames = firstRow
		}
	} else {
		// Explicitly set the number of fields per record to be enforced
		// based on the number of preconfigured column names. Otherwise,
		// csv.Reader would do this implicitly when reading the first row.
		reader.FieldsPerRecord = len(colNames)
	}

	numRowsWithErrors := 0
	defer func() {
		if o.SkipErrors && numRowsWithErrors > 0 {
			basicLog.Printf("Skipped %d lines (rows) due to parsing errors", numRowsWithErrors)
		}
	}()

	allRecords := make([]record, 0)
	for {
		rowFields, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else if o.SkipErrors {
				numRowsWithErrors++
				basicLog.Printf(err.Error())
				continue
			}
			return err
		}

		thisRecord := fieldsToRecord(&colNames, &rowFields)
		allRecords = append(allRecords, thisRecord)
	}

	enc := json.NewEncoder(o.JsonOutput)
	if err := enc.Encode(allRecords); err != nil {
		return err
	}

	return nil
}

// getCsvReader prepares the given io.Reader and returns a new *csv.Reader for parsing its contents as a CSV.
func getCsvReader(r io.Reader) (*csv.Reader, error) {
	// Skip the first rune if it is a BOM
	br := bufio.NewReader(r)
	firstRune, _, err := br.ReadRune()
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
	}
	if firstRune != '\uFEFF' {
		// First rune is not a BOM, so put it back
		br.UnreadRune()
	}

	return csv.NewReader(br), nil
}

// fieldsToRecord creates key/value pairs from column names and row values at corresponding indexes
// in order to populate a record.
func fieldsToRecord(colNames *[]string, rowValues *[]string) record {
	rec := make(record, len(*colNames))

	for i := range *colNames {
		k, v := (*colNames)[i], (*rowValues)[i]
		rec[k] = v
	}

	return rec
}
