package fastxml

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseProcInst(t *testing.T) {
	testCases := []struct {
		Input    string
		Error    string
		Expected Token
		Offset   int
	}{
		{
			Input: `<?target inst?>`,
			Expected: ProcInst{
				Target: []byte("target"),
				Inst:   []byte("inst"),
			},
			Offset: 13,
		},
		{
			Input: `<?invalid?>`,
			Error: "expected ' ' in ProcInst",
		},
		{
			Input: `<?missing end`,
			Error: "expected '?>' to end ProcInst",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Input, func(t *testing.T) {
			actual, offset, err := parseProcInst([]byte(tc.Input[2:]))
			if tc.Error != "" {
				assert.EqualError(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Offset, offset)
				assert.Equal(t, tc.Expected, actual)
			}
		})
	}
}

func Test_parsePotentialDirective(t *testing.T) {
	testCases := []struct {
		Input    string
		Error    string
		Expected Token
		Offset   int
	}{
		{
			Input:    `<![CDATA[raw]]>`,
			Expected: CDATA("raw"),
			Offset:   13,
		},
		{
			Input: "<![CDATA[unterminated",
			Error: "expected ']]>' to end CDATA",
		},
		{
			Input:    `<!--comment-->`,
			Expected: Comment("comment"),
			Offset:   12,
		},
		{
			Input: "<!--unterminated",
			Error: "expected '-->' to end Comment",
		},
		{
			Input:    `<!directive>`,
			Expected: Directive("directive"),
			Offset:   10,
		},
		{
			Input: "<!unterminated",
			Error: "expected '>' to end Directive",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Input, func(t *testing.T) {
			actual, offset, err := parsePotentialDirective([]byte(tc.Input[2:]))
			if tc.Error != "" {
				assert.EqualError(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Offset, offset)
				assert.Equal(t, tc.Expected, actual)
			}
		})
	}
}

func Test_parseElement(t *testing.T) {
	testCases := []struct {
		Input    string
		Error    string
		Expected Token
		Offset   int
		Closed   bool
	}{
		{
			Input: `<xhtml:strong>`,
			Expected: StartElement{
				Name: Name{
					Local: []byte("strong"),
					Space: []byte("xhtml"),
				},
			},
			Offset: 13,
		},
		{
			Input: `<closed/>`,
			Expected: StartElement{
				Name: Name{Local: []byte("closed")},
			},
			Closed: true,
			Offset: 8,
		},
		{
			Input: `<foo key="value" another:complex = "key">`,
			Expected: StartElement{
				Name: Name{Local: []byte("foo")},
				Attr: []Attr{
					Attr{
						Name:  Name{Local: []byte("key")},
						Value: []byte("value"),
					},
					Attr{
						Name: Name{
							Space: []byte("another"),
							Local: []byte("complex"),
						},
						Value: []byte("key"),
					},
				},
			},
			Offset: 40,
		},
		{
			Input: `<unterminated`,
			Error: `expected '>' to end StartElement`,
		},
		{
			Input: `<foo unterminated=>`,
			Error: `expected '"' to start Attr`,
		},
		{
			Input: `<foo unterminated=">`,
			Error: `expected '"' to end Attr`,
		},
		{
			Input: `<foo unterminated>`,
			Error: `unexpected "u" after Attrs`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Input, func(t *testing.T) {
			actual, offset, closed, err := parseElement([]byte(tc.Input[1:]))
			if tc.Error != "" {
				assert.EqualError(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Offset, offset)
				assert.Equal(t, tc.Closed, closed)
				assert.Equal(t, tc.Expected, actual)
			}
		})
	}
}

func TestReader_RawToken(t *testing.T) {
	testCases := []struct {
		Input    string
		Error    string
		Expected []Token
	}{
		{
			Input: `chardata<!--comment-->space<?proc inst?>`,
			Expected: []Token{
				CharData("chardata"),
				Comment("comment"),
				CharData("space"),
				ProcInst{
					Target: []byte("proc"),
					Inst:   []byte("inst"),
				},
			},
		},
		{
			Input: `<foo key="value" />extra`,
			Expected: []Token{
				StartElement{
					Name: Name{Local: []byte("foo")},
					Attr: []Attr{
						Attr{
							Name:  Name{Local: []byte("key")},
							Value: []byte("value"),
						},
					},
				},
				EndElement{
					Name: Name{Local: []byte("foo")},
				},
				CharData("extra"),
			},
		},
		{
			Input: `invalid<>`,
			Error: "not enough bytes (1) remaining for valid XML element declaration",
		},
		{
			Input: `<invalid`,
			Error: "expected '>' to end StartElement",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Input, func(t *testing.T) {
			r := NewTokenReader([]byte(tc.Input))
			var actual []Token
			var err error
			for {
				var token Token
				token, err = r.RawToken()
				if err != nil {
					if err == io.EOF {
						err = nil
					}
					break
				}
				actual = append(actual, token)
			}
			if tc.Error != "" {
				assert.EqualError(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, actual)
				assert.True(t, r.InputOffset() != 0)
			}
		})
	}
}
