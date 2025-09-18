package assoc_test

import (
	"testing"

	"github.com/TylerHendrickson/csv2json/assoc"
	"github.com/stretchr/testify/assert"
)

func TestSlicesToMap(t *testing.T) {
	keys := []string{"a", "b", "c"}
	for _, tt := range []struct {
		name     string
		values   []string
		expected map[string]string
		err      error
	}{
		{
			"equal lengths",
			[]string{"cat", "dog", "goldfish"},
			map[string]string{"a": "cat", "b": "dog", "c": "goldfish"},
			nil,
		},
		{
			"more keys than values",
			[]string{"mouse", "rat"},
			map[string]string{"a": "mouse", "b": "rat", "c": ""},
			assoc.ErrValuesFewerThanKeys,
		},
		{
			"more values than keys",
			[]string{"humpback", "beluga", "blue", "killer"},
			map[string]string{"a": "humpback", "b": "beluga", "c": "blue"},
			assoc.ErrValuesExceedKeys,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, err := assoc.ZipToMap(keys, tt.values)
			if tt.err != nil {
				// assert.EqualError(t, tt.err, err.Error())
				assert.ErrorIs(t, err, tt.err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, m)
		})
	}
}
