package main

import (
	"encoding/csv"
	"encoding/json"
	"github.com/integrii/flaggy"
	"io"
	"log"
	"os"
)

// record values are a single row's worth of data, keyed by column names
type record map[string]string

func main() {
	var colNames []string
	var fileName string

	flaggy.SetVersion("0.1.0")
	flaggy.SetDescription("Restructures CSV into JSON")
	flaggy.StringSlice(&colNames, "c", "force-columns",
		"Column names, which must equal the number of CSV fields if given. "+
			"When set, the first line of CSV data is treated as a data row instead of column names.")
	flaggy.AddPositionalValue(&fileName, "file", 1, false,
		"The CSV file to convert. If omitted, input is read from stdin.")
	flaggy.Parse()

	fd, err := getCsvFile(fileName)
	if err != nil {
		log.Fatalln(err)
	}

	if err := csv2Json(colNames, fd, os.Stdout); err != nil {
		log.Fatalln(err)
	}
}

// csv2Json converts CSV data from io.Reader to a JSON array and emits the result
// to io.Writer. When `colNames` is empty, headers are derived from the first
// line of the CSV file. Returns any errors from reading CSV or encoding JSON.
func csv2Json(colNames []string, input io.Reader, out io.Writer) error {
	reader := csv.NewReader(input)

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

	allRecords := make([]record, 0)
	for {
		rowFields, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		//thisRecord := make(record, reader.FieldsPerRecord)
		thisRecord := fieldsToRecord(&colNames, &rowFields)
		allRecords = append(allRecords, thisRecord)
	}

	enc := json.NewEncoder(out)
	if err := enc.Encode(allRecords); err != nil {
		return err
	}

	return nil
}

// getCsvFile gets a pointer to an open os.File named by filename, or else
// os.Stdin. Errors encountered when opening named files are propagated from os.Open().
func getCsvFile(fileName string) (*os.File, error) {
	if fileName == "" {
		return os.Stdin, nil
	} else {
		return os.Open(fileName)
	}
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
