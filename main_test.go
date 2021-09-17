package main

import (
	"bytes"
	"encoding/csv"
	"github.com/integrii/flaggy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestCli(t *testing.T) {
	for _, tt := range []struct {
		testName      string
		useNamedInput bool
		useTempFile   bool
		csv           string
		wantJsonOut   string
		wantErr       error
		cliArgs       []string
	}{
		{
			"Basic conversion from named input",
			true,
			true,
			"a,b,c\n1,2,3\nz,y,x\n",
			`[{"a": "1", "b": "2", "c": "3"}, {"a": "z", "b": "y", "c": "x"}]`,
			nil,
			[]string{},
		},
		{
			"Basic conversion from stdin",
			false,
			true,
			"a,b,c\n1,2,3\nz,y,x\n",
			`[{"a": "1", "b": "2", "c": "3"}, {"a": "z", "b": "y", "c": "x"}]`,
			nil,
			[]string{},
		},
		{
			"Fails when missing file is named",
			true,
			false,
			"a,b,c\n1,2,3\nz,y,x\n",
			"",
			os.ErrNotExist,
			[]string{},
		},
		{
			"Fails when CSV field count",
			true,
			true,
			"a,b,c\n1,2\nz,y,x\n",
			"",
			csv.ErrFieldCount,
			[]string{},
		},
		{
			"Cannot skip CSV parse error in header line",
			true,
			true,
			"a,\"b,c\n1,2,3\nz,y,x\n",
			"",
			csv.ErrQuote,
			[]string{"--skip-errors"},
		},
		{
			"Can skip CSV parse error in non-header line",
			true,
			true,
			"a,b,c\n1,\"\"2',3\nz,y,x\n",
			`[{"a": "z", "b": "y", "c": "x"}]`,
			nil,
			[]string{"--skip-errors"},
		},
	} {
		t.Run(tt.testName, func(t *testing.T) {

			tempFile, err := ioutil.TempFile("", "csv2json-test-*")
			require.NoError(t, err, "Test cannot run without a temp file")

			if tt.useTempFile {
				_, err = tempFile.WriteString(tt.csv)
				require.NoError(t, err, "Test cannot run without populated CSV")
				_, err = tempFile.Seek(0, 0)
				require.NoError(t, err, "Test cannot run without CSV ready for reading")
				require.FileExists(t, tempFile.Name(), "Test cannot run without a temp file")
				t.Cleanup(func() {
					if err := os.Remove(tempFile.Name()); err != nil {
						t.Fatalf("Error removing tempfile during cleanup")
					}
				})
			} else {
				if err := os.Remove(tempFile.Name()); err != nil {
					t.Fatalf("Error removing tempfile during setup")
				}
			}

			cliArgs := append([]string{"csv2json"}, tt.cliArgs...)
			if tt.useNamedInput {
				// Provide the CSV temp file as positional CLI argument
				cliArgs = append(cliArgs, tempFile.Name())
			} else {
				// Substitute CSV temp file for stdin
				oldStdIn := os.Stdin
				os.Stdin = tempFile
				t.Cleanup(func() {
					// Restore stdin
					os.Stdin = oldStdIn
				})
			}
			os.Args = cliArgs
			flaggy.ResetParser()

			// Capture stdout for JSON assertions and suppress logging to stderr
			capturedStdout, runErr := func() (string, error) {
				oldStdout := os.Stdout
				oldLogOutput := log.Writer()
				defer func() {
					// Restore stdout and resume logging to stderr
					os.Stdout = oldStdout
					log.SetOutput(oldLogOutput)
				}()
				stdoutReader, stdoutWriter, _ := os.Pipe()
				os.Stdout = stdoutWriter
				log.SetOutput(bytes.NewBuffer([]byte{})) // Discard logs sent to stderr

				runErr := runCli()

				stdoutWriter.Close()
				var buf bytes.Buffer
				io.Copy(&buf, stdoutReader)
				return buf.String(), runErr
			}()

			if tt.wantErr == nil {
				assert.NoError(t, runErr)
				assert.JSONEq(t, tt.wantJsonOut, capturedStdout)
			} else {
				assert.ErrorIs(t, runErr, tt.wantErr)
			}
		})
	}
}

func TestCsv2Json(t *testing.T) {
	// Log errors to nowhere while this test runs
	oldLogOutput := log.Writer()
	log.SetOutput(bytes.NewBuffer([]byte{}))
	t.Cleanup(func() {
		log.SetOutput(oldLogOutput)
	})

	for _, tt := range []struct {
		testName      string
		forceColumns  []string
		csv           string
		wantJson      string
		skipErrors    bool
		expectedError error
	}{
		{
			"Basic conversion with columns from first row",
			[]string{},
			"a,b,c\n1,2,3\nz,y,x\n",
			`[{"a": "1", "b": "2", "c": "3"}, {"a": "z", "b": "y", "c": "x"}]`,
			false,
			nil,
		},
		{
			"Whitespace CSV converts to empty JSON array",
			[]string{},
			"    ",
			`[]`,
			false,
			nil,
		},
		{
			"Empty CSV converts to empty JSON array",
			[]string{},
			"",
			`[]`,
			false,
			nil,
		},
		{
			"Empty CSV with forced columns converts to empty JSON array",
			[]string{"a", "b", "c"},
			"",
			`[]`,
			false,
			nil,
		},
		{
			"Basic conversion with forced columns",
			[]string{"a", "b", "c"},
			"alpha,bravo,charlie\n1,2,3\nz,y,x\n",
			`[{"a": "alpha", "b": "bravo", "c": "charlie"}, {"a": "1", "b": "2", "c": "3"}, {"a": "z", "b": "y", "c": "x"}]`,
			false,
			nil,
		},
		{
			"Errors abort conversion",
			[]string{},
			"a,b,c\n1,2,3\nbad,line\nz,y,x\n",
			`[{"a": "1", "b": "2", "c": "3"}, {"a": "z", "b": "y", "c": "x"}]`,
			false,
			&csv.ParseError{
				StartLine: 3,
				Line:      3,
				Column:    0,
				Err:       csv.ErrFieldCount,
			},
		},
		{
			"Errors can be skipped",
			[]string{},
			"a,b,c\n1,2,3\nbad,line\nz,y,x\n",
			`[{"a": "1", "b": "2", "c": "3"}, {"a": "z", "b": "y", "c": "x"}]`,
			true,
			nil,
		},
	} {
		t.Run(tt.testName, func(t *testing.T) {
			jsonStream := bytes.NewBuffer([]byte{})
			options := conversionOptions{
				colNames:   tt.forceColumns,
				csvInput:   bytes.NewReader([]byte(tt.csv)),
				jsonOutput: jsonStream,
				skipErrors: tt.skipErrors,
			}

			err := csv2Json(options)

			if tt.expectedError == nil {
				assert.NoError(t, err)
				assert.JSONEq(t, tt.wantJson, jsonStream.String())
			} else if !tt.skipErrors {
				assert.EqualError(t, err, tt.expectedError.Error())
			}
		})
	}
}

func TestGetCsvFile(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "csv2json-test-*")
	require.NoError(t, err, "Tests cannot run without a temp file")
	require.FileExists(t, tempFile.Name(), "Tests cannot run without a temp file")
	t.Cleanup(func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Fatalf("Error removing tempfile during cleanup")
		}
	})
	testBadFileName := "thisFileDoesNotExist"
	require.NoFileExists(t, testBadFileName, "Tests cannot run if this file exists")

	for _, tt := range []struct {
		testName, testFileName string
		wantFd                 *os.File
	}{
		{"Gets named file", tempFile.Name(), tempFile},
		{"Gets stdin when no named file", "", os.Stdin},
		{"Error when named file does not exist", testBadFileName, nil},
	} {
		t.Run(tt.testName, func(t *testing.T) {
			fd, err := getCsvFile(tt.testFileName)

			if tt.wantFd != nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantFd.Name(), fd.Name())
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestFieldsToRecords(t *testing.T) {
	for _, tt := range []struct {
		testName            string
		colNames, rowValues []string
		wantRecord          record
	}{
		{
			"Values keyed by column name at same index",
			[]string{"a", "b", "c"}, []string{"1", "2", "3", "4"},
			record{"a": "1", "b": "2", "c": "3"},
		},
		{
			"Values with no same-index column are excluded from result",
			[]string{"a", "b"}, []string{"1", "2", "3"},
			record{"a": "1", "b": "2"},
		},
		{
			"No columns result in empty record",
			[]string{}, []string{"1", "2", "3"},
			record{},
		},
	} {
		t.Run(tt.testName, func(t *testing.T) {
			gotRecord := fieldsToRecord(&tt.colNames, &tt.rowValues)

			assert.Equal(t, tt.wantRecord, gotRecord)
		})
	}
}
