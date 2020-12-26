package fastxml

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsElement(t *testing.T) {
	assert.True(t, IsElement([]byte("<text>")))
	assert.False(t, IsElement([]byte("<!text>")))
	assert.False(t, IsElement([]byte("<?proc inst>")))
}

func TestIsSelfClosing(t *testing.T) {
	assert.True(t, IsSelfClosing([]byte("<text/>")))
	assert.False(t, IsSelfClosing([]byte("<text>")))
}

func TestIsEndElement(t *testing.T) {
	assert.True(t, IsEndElement([]byte("</text>")))
	assert.False(t, IsEndElement([]byte("<text>")))
}

func TestIsStartElement(t *testing.T) {
	assert.False(t, IsStartElement([]byte("</text>")))
	assert.True(t, IsStartElement([]byte("<text>")))
}

func TestElement(t *testing.T) {
	testCases := []struct {
		Token string
		Name  string
		Attrs string
	}{
		{
			Token: `<start>`,
			Name:  "start",
		},
		{
			Token: `<start/>`,
			Name:  "start",
		},
		{
			Token: `</end>`,
			Name:  "end",
		},
		{
			Token: `<foo key="val">`,
			Name:  "foo",
			Attrs: `key="val"`,
		},
		{
			Token: `<foo key="val"/>`,
			Name:  "foo",
			Attrs: `key="val"`,
		},
		{
			Token: `<foo key="val" />`,
			Name:  "foo",
			Attrs: `key="val" `,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Token, func(t *testing.T) {
			name, attrs := Element([]byte(tc.Token))
			assert.Equal(t, tc.Name, string(name))
			if tc.Attrs == "" {
				assert.Equal(t, []byte(nil), attrs)
			} else {
				assert.Equal(t, tc.Attrs, string(attrs))
			}
		})
	}
}

func TestAttrs(t *testing.T) {
	testCases := []struct {
		Token string
		Key   []string
		Value []string
		Error string
		Limit int
	}{
		{
			Token: ``,
			Error: "",
		},
		{
			Token: `key="value"`,
			Key:   []string{"key"},
			Value: []string{"value"},
		},
		{
			Token: `key="value"  extraspace = " val2" `,
			Key:   []string{"key", "extraspace"},
			Value: []string{"value", " val2"},
		},
		{
			Token: `key="value" anotherkey="val"`,
			Limit: 1,
			Error: "terminated",
		},
		{
			Token: `key`,
			Error: "expected whitespace but got non-whitespace",
		},
		{
			Token: `key=`,
			Error: `expected Attr to start with '"'`,
		},
		{
			Token: `key="`,
			Error: `expected Attr to end with '"'`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Token, func(t *testing.T) {
			var keys []string
			var vals []string
			err := Attrs([]byte(tc.Token), func(key, val []byte) error {
				keys = append(keys, string(key))
				vals = append(vals, string(val))
				if len(keys) == tc.Limit {
					return errors.New("terminated")
				}
				return nil
			})
			if tc.Error != "" {
				assert.EqualError(t, err, tc.Error)
			} else {
				assert.Equal(t, tc.Key, keys)
				assert.Equal(t, tc.Value, vals)
			}
		})
	}
}
