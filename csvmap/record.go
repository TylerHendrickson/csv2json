package csvmap

import "github.com/TylerHendrickson/csv2json/assoc"

// Record is a single CSV row keyed by field name.
type Record = map[string]string

// MapRowValues is syntactic sugar for calling assoc.ZipToMap in the context of mapping
// CSV field names and values.
func MapRowValues(fieldNames, fieldValues []string) (Record, error) {
	return assoc.ZipToMap(fieldNames, fieldValues)
}
