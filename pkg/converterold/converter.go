package converterold

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
)

// record values are a single row's worth of data, keyed by column names
type record map[string]string

type Options struct {
	ColNames   []string
	CsvInput   io.Reader
	JsonOutput io.Writer
	SkipErrors bool
}

func (o *Options) C () error {
	reader := csv.NewReader(o.CsvInput)

	colNames := o.ColNames
	if len(colNames) == 0 {
		// Read the first line to get column names
		firstRow, err := reader.Read()
		if err != nil {
			if err != io.EOF {
				return err
			}
		} else {
			colNames = firstRow
		}
	} else {
		reader.FieldsPerRecord = len(colNames)
	}

	numRowsWithErrors := 0
	defer func() {
		if o.SkipErrors && numRowsWithErrors > 0 {
			log.Printf("Skipped %d lines (rows) due to parsing errors", numRowsWithErrors)
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
				log.Printf(err.Error())
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

// Convert converts CSV data from io.Reader to a JSON array and emits the result
// to io.Writer. When `colNames` is empty, headers are derived from the first
// line of the CSV file. Returns any errors from reading CSV or encoding JSON.
func Convert(options Options) error {
	reader := csv.NewReader(options.CsvInput)

	colNames := options.ColNames
	if len(colNames) == 0 {
		// Read the first line to get column names
		firstRow, err := reader.Read()
		if err != nil {
			if err != io.EOF {
				return err
			}
		} else {
			colNames = firstRow
		}
	} else {
		reader.FieldsPerRecord = len(colNames)
	}

	numRowsWithErrors := 0
	defer func() {
		if options.SkipErrors && numRowsWithErrors > 0 {
			log.Printf("Skipped %d lines (rows) due to parsing errors", numRowsWithErrors)
		}
	}()

	allRecords := make([]record, 0)
	for {
		rowFields, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else if options.SkipErrors {
				numRowsWithErrors++
				log.Printf(err.Error())
				continue
			}
			return err
		}

		thisRecord := fieldsToRecord(&colNames, &rowFields)
		allRecords = append(allRecords, thisRecord)
	}

	enc := json.NewEncoder(options.JsonOutput)
	if err := enc.Encode(allRecords); err != nil {
		return err
	}

	return nil
}

// fieldsToRecord creates key/value pairs from column names and row values at
// corresponding indexes in order to populate a record.
func fieldsToRecord(colNames *[]string, rowValues *[]string) record {
	rec := make(record, len(*colNames))

	for i := range *colNames {
		k, v := (*colNames)[i], (*rowValues)[i]
		rec[k] = v
	}

	return rec
}
