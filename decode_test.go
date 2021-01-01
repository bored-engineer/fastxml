package fastxml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeEntities(t *testing.T) {
	testCases := []struct {
		Input    string
		Error    string
		Expected string
	}{
		{
			Input:    `Hello World`,
			Expected: `Hello World`,
		},
		{
			Input:    `Fast&amp;&quot;&apos;&gt;&lt;Path`,
			Expected: `Fast&"'><Path`,
		},
		{
			Input:    `It costs &pound;1`,
			Expected: `It costs £1`,
		},
		{
			Input:    `&#x00A9; 2020`,
			Expected: `© 2020`,
		},
		{
			Input:    `1 &#60; 2`,
			Expected: `1 < 2`,
		},
		{
			Input: `&#1234567891011;`,
			Error: `failed to decode "1234567891011": strconv.ParseInt: parsing "1234567891011": value out of range`,
		},
		{
			Input: `&#xnothex;`,
			Error: `failed to decode "nothex": strconv.ParseInt: parsing "nothex": invalid syntax`,
		},
		{
			Input: `&`,
			Error: `expected ';' to end XML entity, not found`,
		},
		{
			Input: `&invalid;`,
			Error: `unknown XML entity "invalid"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Input, func(t *testing.T) {
			actual, err := DecodeEntities([]byte(tc.Input), nil)
			if tc.Error != "" {
				assert.EqualError(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, string(actual))
			}
			prepend := []byte("prepend")
			actual, err = DecodeEntitiesAppend(prepend, []byte(tc.Input))
			assert.Equal(t, []byte("prepend"), actual[:len(prepend)])
			if tc.Error != "" {
				assert.EqualError(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, string(actual[len(prepend):]))
			}
		})
	}
}
