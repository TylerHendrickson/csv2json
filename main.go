package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"github.com/integrii/flaggy"
	"io"
	"log"
	"os"
)

// record values are a single row's worth of data, keyed by column names
type record map[string]string

type conversionOptions struct {
	colNames   []string
	csvInput   io.Reader
	jsonOutput io.Writer
	skipErrors bool
}

func main() {
	if err := runCli(); err != nil {
		log.Fatalln(err)
	}
}

func runCli() error {
	var (
		colNames   []string
		fileName   string
		skipErrors bool
	)
	flaggy.SetVersion("0.2.0")
	flaggy.SetDescription("Restructures CSV into JSON")
	flaggy.StringSlice(&colNames, "c", "force-columns",
		"Column names, which must equal the number of CSV fields if given. "+
			"When set, the first line of CSV data is treated as a data row instead of column names.")
	flaggy.AddPositionalValue(&fileName, "file", 1, false,
		"The CSV file to convert. If omitted, input is read from stdin.")
	flaggy.Bool(&skipErrors, "s", "skip-errors",
		"Skip CSV lines that cause parsing errors. By default, errors abort conversion completely.")
	flaggy.Parse()

	fd, err := getCsvFile(fileName)
	if err != nil {
		return err
	}

	options := conversionOptions{
		colNames:   colNames,
		csvInput:   fd,
		jsonOutput: os.Stdout,
		skipErrors: skipErrors,
	}

	if err := csv2Json(options); err != nil {
		return err
	}

	return nil
}

// csv2Json converts CSV data from io.Reader to a JSON array and emits the result
// to io.Writer. When `colNames` is empty, headers are derived from the first
// line of the CSV file. Returns any errors from reading CSV or encoding JSON.
func csv2Json(options conversionOptions) error {
	reader := getCsvReader(options.csvInput)

	colNames := options.colNames
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
		if options.skipErrors && numRowsWithErrors > 0 {
			log.Printf("Skipped %d lines (rows) due to parsing errors", numRowsWithErrors)
		}
	}()

	allRecords := make([]record, 0)
	for {
		rowFields, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else if options.skipErrors {
				numRowsWithErrors++
				log.Printf(err.Error())
				continue
			}
			return err
		}

		thisRecord := fieldsToRecord(&colNames, &rowFields)
		allRecords = append(allRecords, thisRecord)
	}

	enc := json.NewEncoder(options.jsonOutput)
	if err := enc.Encode(allRecords); err != nil {
		return err
	}

	return nil
}

// getCsvFile gets a pointer to an open os.File named by filename, or else
// os.Stdin. Errors encountered when opening named files are propagated from
// os.Open().
func getCsvFile(fileName string) (*os.File, error) {
	if fileName == "" {
		return os.Stdin, nil
	} else {
		return os.Open(fileName)
	}
}

// getCsvReader prepares the given io.Reader and returns a new *csv.Reader for parsing its contents as a CSV.
func getCsvReader(r io.Reader) *csv.Reader {
	// Skip the first rune if it is a BOM
	br := bufio.NewReader(r)
	firstRune, _, err := br.ReadRune()
	if  err != nil {
		if err != io.EOF {
			log.Fatal(err)
		}
	}
	if  firstRune != '\uFEFF' {
		// First rune is not a BOM, so put it back
		br.UnreadRune()
	}

	return csv.NewReader(br)
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
