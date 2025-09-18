package assoc_test

import (
	"io"
	"testing"

	"github.com/TylerHendrickson/csv2json/assoc"
	"github.com/stretchr/testify/assert"
)

type testSliceReader struct {
	idx     int
	records [][]string
}

func (r *testSliceReader) Read() ([]string, error) {
	if r.idx >= len(r.records) {
		return nil, io.EOF
	}
	record := r.records[r.idx]
	r.idx++
	return record, nil
}

func TestSliceReaderMapper(t *testing.T) {
	var (
		records = [][]string{{"a", "b", "c"}, {"d", "e", "f"}}
		rec     map[string]string
		err     error
	)
	t.Run("just right", func(t *testing.T) {
		sm := assoc.NewMapReader(&testSliceReader{0, records}, []string{"zero", "one", "two"})

		rec, err = sm.Read()
		assert.Equal(t, rec, map[string]string{"zero": "a", "one": "b", "two": "c"})
		assert.NoError(t, err)

		rec, err = sm.Read()
		assert.Equal(t, rec, map[string]string{"zero": "d", "one": "e", "two": "f"})
		assert.NoError(t, err)

		rec, err = sm.Read()
		assert.Nil(t, rec)
		assert.EqualError(t, err, io.EOF.Error())
	})

	t.Run("more keys than values", func(t *testing.T) {
		sm := assoc.NewMapReader(&testSliceReader{0, records}, []string{"zero", "one", "two", "three"})

		rec, err = sm.Read()
		assert.Equal(t, rec, map[string]string{"zero": "a", "one": "b", "two": "c", "three": ""})
		assert.EqualError(t, err, assoc.ErrValuesFewerThanKeys.Error())

		rec, err = sm.Read()
		assert.Equal(t, rec, map[string]string{"zero": "d", "one": "e", "two": "f", "three": ""})
		assert.EqualError(t, err, assoc.ErrValuesFewerThanKeys.Error())

		rec, err = sm.Read()
		assert.Nil(t, rec)
		assert.EqualError(t, err, io.EOF.Error())
	})

	t.Run("more values than keys", func(t *testing.T) {
		sm := assoc.NewMapReader(&testSliceReader{0, records}, []string{"zero", "one"})

		rec, err = sm.Read()
		assert.Equal(t, rec, map[string]string{"zero": "a", "one": "b"})
		assert.EqualError(t, err, assoc.ErrValuesExceedKeys.Error())

		rec, err = sm.Read()
		assert.Equal(t, rec, map[string]string{"zero": "d", "one": "e"})
		assert.EqualError(t, err, assoc.ErrValuesExceedKeys.Error())

		rec, err = sm.Read()
		assert.Nil(t, rec)
		assert.EqualError(t, err, io.EOF.Error())
	})
}
