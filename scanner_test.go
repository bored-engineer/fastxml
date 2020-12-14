package fastxml

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanner_Skip(t *testing.T) {
	s := NewScanner([]byte(`<nested><element>with data</element><closing/><?skip me></nested>more`))
	// Read <nested>
	token, chardata, err := s.Next()
	assert.NoError(t, err)
	assert.Equal(t, false, chardata)
	assert.Equal(t, []byte("<nested>"), token)
	// Skip children
	err = s.Skip()
	assert.NoError(t, err)
	// Read final "more"
	token, chardata, err = s.Next()
	assert.NoError(t, err)
	assert.Equal(t, true, chardata)
	assert.Equal(t, []byte("more"), token)
	// EOF
	_, _, err = s.Next()
	assert.Equal(t, io.EOF, err)
	// Verify error
	s.Reset([]byte("<?invalid"))
	err = s.Skip()
	assert.Error(t, err)
}

func TestScanner(t *testing.T) {
	type result struct {
		Token    []byte
		CharData bool
		Offset   int
	}
	testCases := []struct {
		Input    string
		Error    string
		Expected []result
	}{
		{
			Input:    ``,
			Expected: []result(nil),
		}, {
			Input: `foo`,
			Expected: []result{{
				Token:    []byte("foo"),
				CharData: true,
			}},
		}, {
			Input: `<![CDATA[nested<xml>]]>`,
			Expected: []result{{
				Token:    []byte(`<![CDATA[nested<xml>]]>`),
				CharData: true,
			}},
		}, {
			Input: `foo<bar><gar /></bar><![CDATA[test]]>har`,
			Expected: []result{
				{
					Offset:   0,
					Token:    []byte(`foo`),
					CharData: true,
				}, {
					Offset: 3,
					Token:  []byte(`<bar>`),
				}, {
					Offset: 8,
					Token:  []byte(`<gar />`),
				}, {
					Offset: 15,
					Token:  []byte(`</bar>`),
				}, {
					Offset:   21,
					Token:    []byte(`<![CDATA[test]]>`),
					CharData: true,
				}, {
					Offset:   37,
					Token:    []byte(`har`),
					CharData: true,
				},
			},
		}, {
			Input: `<unterminated`,
			Error: `expected Token to end with '>'`,
		}, {
			Input: `<![CDATA[unterminated`,
			Error: `expected Token to end with ']]>'`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Input, func(t *testing.T) {
			s := NewScanner([]byte(tc.Input))
			var actual []result
			var err error
			for {
				offset := s.Offset()
				token, chardata, nErr := s.Next()
				if len(token) > 0 {
					actual = append(actual, result{
						Token:    token,
						CharData: chardata,
						Offset:   offset,
					})
				}
				if nErr != nil {
					if nErr != io.EOF {
						err = nErr
					}
					break
				}
			}
			if tc.Error != "" {
				assert.EqualError(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, actual)
			}
		})
	}
}

func BenchmarkScanner(b *testing.B) {
	data := benchData(b)
	for n := 0; n < b.N; n++ {
		d := NewScanner(data)
		for {
			_, _, err := d.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	}
}
