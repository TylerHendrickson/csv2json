package assoc

// SliceReader is an interface for progressively reading slices,
// e.g. rows of values in a CSV file.
// SliceReaders may encounter an error, such as EOF, when reading its source for the next slice.
type SliceReader[T any] interface {
	// Read returns the next slice, if any.
	Read() ([]T, error)
}

// MapReader is an interface for reading slices and mapping them to known keys,
// e.g. a row of CSV data values keyed by their corresponding field name.
type MapReader[K comparable, V any] interface {
	Read() (map[K]V, error)
}

// mapReader contains a slice of reusable map keys (e.g. CSV columns),
// as well as a SliceReader which acts as a source of values (e.g. CSV rows) for those maps.
type mapReader[K comparable, V any] struct {
	keys   []K
	reader SliceReader[V]
}

// NewMapReader creates a new slice mapper for a slice of keys and SliceReader.
func NewMapReader[K comparable, V any](reader SliceReader[V], keys []K) MapReader[K, V] {
	return &mapReader[K, V]{keys, reader}
}

// Read reads the next slice from SliceReader and returns a new map by zipping together
// members of the SliceReaderMapper's `keys` slice with members of the just-read slice.
func (sm *mapReader[K, V]) Read() (map[K]V, error) {
	recordValues, err := sm.reader.Read()
	if err != nil {
		return nil, err
	}
	return ZipToMap(sm.keys, recordValues)
}
