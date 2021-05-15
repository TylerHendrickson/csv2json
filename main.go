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

var fileName string
var colNames []string

func init() {
	flaggy.SetVersion("0.1.0")
	flaggy.SetDescription("Restructures CSV into JSON")
	flaggy.StringSlice(&colNames, "c", "columns",
		"Column names, which must equal the number of CSV fields if given. "+
			"When set, the first line of CSV data is treated as a data row instead of column names.")
	flaggy.AddPositionalValue(&fileName, "file", 1, false,
		"The CSV file to convert. If omitted, input is read from stdin.")
	flaggy.Parse()
}

func main() {
	fd, err := getCsvFile()
	if err != nil {
		log.Fatalln(err)
	}

	csv2Json(fd)
}

// csv2Json reads CSV data from os.File fd and converts it to a JSON array, which is emitted to os.Stdout
func csv2Json(fd *os.File) {
	reader := csv.NewReader(fd)

	if len(colNames) == 0 {
		// Read the first line to get column names
		firstRow, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				return
			} else {
				log.Fatalln(err)
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
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalln(err)
		}

		thisRecord := make(record, reader.FieldsPerRecord)
		fieldsToRecord(&colNames, &rowFields, &thisRecord)
		allRecords = append(allRecords, thisRecord)
	}

	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(allRecords); err != nil {
		log.Fatalln(err)
	}
}

// getCsvFile returns a file descriptor pointer for a named file, if given, or else os.Stdin.
// Errors opening named files are propagated from os.Open().
func getCsvFile() (*os.File, error) {
	if fileName == "" {
		return os.Stdin, nil
	} else {
		return os.Open(fileName)
	}
}

// fieldsToRecord creates key/value pairs from column names and row values at corresponding indexes
// in order to populate a record.
func fieldsToRecord(colNames *[]string, rowValues *[]string, r *record) {
	for i := range *colNames {
		k, v := (*colNames)[i], (*rowValues)[i]
		(*r)[k] = v
	}
}
