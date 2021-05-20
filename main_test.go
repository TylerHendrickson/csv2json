package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func TestCsv2Json(t *testing.T) {
	for _, tt := range []struct {
		testName     string
		forceColumns []string
		csv          string
		wantJson     string
	}{
		{
			"Basic conversion with columns from first row",
			[]string{},
			"a,b,c\n1,2,3\nz,y,x\n",
			`[{"a": "1", "b": "2", "c": "3"}, {"a": "z", "b": "y", "c": "x"}]`,
		},
		{
			"Whitespace CSV converts to empty JSON array",
			[]string{},
			"    ",
			`[]`,
		},
		{
			"Empty CSV converts to empty JSON array",
			[]string{},
			"",
			`[]`,
		},
		{
			"Empty CSV with forced columns converts to empty JSON array",
			[]string{"a", "b", "c"},
			"",
			`[]`,
		},
		{
			"Basic conversion with forced columns",
			[]string{"a", "b", "c"},
			"alpha,bravo,charlie\n1,2,3\nz,y,x\n",
			`[{"a": "alpha", "b": "bravo", "c": "charlie"}, {"a": "1", "b": "2", "c": "3"}, {"a": "z", "b": "y", "c": "x"}]`,
		},
	} {
		t.Run(tt.testName, func(t *testing.T) {
			csvReader := bytes.NewReader([]byte(tt.csv))
			jsonWriter := bytes.NewBuffer([]byte{})

			err := csv2Json(tt.forceColumns, csvReader, jsonWriter)

			assert.NoError(t, err)
			assert.JSONEq(t, tt.wantJson, jsonWriter.String())
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
