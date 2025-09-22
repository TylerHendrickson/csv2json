package assoc

import "errors"

var (
	ErrValuesExceedKeys    = errors.New("values length exceeds keys length")
	ErrValuesFewerThanKeys = errors.New("values length fewer than keys length")
)

// ZipToMap produces a map by associating (or "zipping") same-index pairs of "keys" and "values" slice members,
// (e.g. column names and values from a row).
// The length of the returned map is determined by the length of the keys slice.
// If there are more keys than values, keys with no corresponding value will be mapped to zero-value of V
// and the returned error will be ErrValuesFewerThanKeys.
// If there are more values than keys, values without a corresponding key will not be mapped,
// and the returned error will be ErrValuesExceedKeys.
func ZipToMap[K comparable, V any](keys []K, values []V) (map[K]V, error) {
	var err error
	if numKeys, numValues := len(keys), len(values); numValues < numKeys {
		err = ErrValuesFewerThanKeys
		// Pad values slice with zero-value members
		values = expandSlice(values, numKeys)
	} else if numValues > numKeys {
		err = ErrValuesExceedKeys
		// Constrain values slice length to that of keys slice
		values = values[:numKeys]
	}
	return zipToMap(keys, values), err
}

// expandSlice pads the given slice to length n with zero-value members
func expandSlice[T any](s []T, n int) []T {
	newSlice := make([]T, n)
	copy(newSlice, s)
	return newSlice
}

// zipToMap pairs ("zips") keys and values items at the same index to produce a map.
func zipToMap[K comparable, V any](keys []K, values []V) map[K]V {
	record := make(map[K]V, len(keys))
	for i, k := range keys {
		record[k] = values[i]
	}
	return record
}
