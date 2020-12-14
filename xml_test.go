package fastxml

import (
	"bytes"
	"encoding/xml"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXMLToken(t *testing.T) {
	testCases := []struct {
		Token    string
		Chardata bool
		Error    string
		Expected xml.Token
	}{
		{
			Token:    `Hello World`,
			Chardata: true,
			Expected: xml.CharData([]byte(`Hello World`)),
		},
		{
			Token: `<true>`,
			Expected: xml.StartElement{
				Name: xml.Name{Local: "true"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Token, func(t *testing.T) {
			actual, err := XMLToken([]byte(tc.Token), tc.Chardata)
			if tc.Error != "" {
				assert.EqualError(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, actual)
			}
		})
	}
}

func TestXMLTokenReader(t *testing.T) {
	testCases := []struct {
		Input    string
		Error    string
		Expected []xml.Token
	}{
		{
			Input: `<hello key="v&amp;lue">world</hello>`,
			Expected: []xml.Token{
				xml.StartElement{
					Name: xml.Name{Local: "hello"},
					Attr: []xml.Attr{
						xml.Attr{
							Name:  xml.Name{Local: "key"},
							Value: "v&lue",
						},
					},
				},
				xml.CharData("world"),
				xml.EndElement{
					Name: xml.Name{Local: "hello"},
				},
			},
		},
		{
			Input: "<parent><child/></parent>",
			Expected: []xml.Token{
				xml.StartElement{
					Name: xml.Name{Local: "parent"},
				},
				xml.StartElement{
					Name: xml.Name{Local: "child"},
				},
				xml.EndElement{
					Name: xml.Name{Local: "child"},
				},
				xml.EndElement{
					Name: xml.Name{Local: "parent"},
				},
			},
		},
		{
			Input: "<?proc inst?>then<!directive>with<![CDATA[some data]]><!--comment-->",
			Expected: []xml.Token{
				xml.ProcInst{
					Target: "proc",
					Inst:   []byte("inst"),
				},
				xml.CharData("then"),
				xml.Directive("directive"),
				xml.CharData("with"),
				xml.CharData("some data"),
				xml.Comment("comment"),
			},
		},
		{
			Input: "<?invalid",
			Error: "expected Token to end with '>'",
		},
		{
			Input: "&invalid;",
			Error: `unknown XML entity "invalid"`,
		},
		{
			Input: `<element key="&invalid;">`,
			Error: `unknown XML entity "invalid"`,
		},
		{
			Input: `<element key="invalid>`,
			Error: `expected Attr to end with '"'`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Input, func(t *testing.T) {
			r := NewXMLTokenReader(NewScanner([]byte(tc.Input)))
			var err error
			var tokens []xml.Token
			for {
				var token xml.Token
				token, err = r.Token()
				if token != nil {
					tokens = append(tokens, token)
				}
				if err != nil {
					if err == io.EOF {
						err = nil
					}
					break
				}
			}
			if tc.Error != "" {
				t.Log(tc.Input, tokens, err)
				assert.EqualError(t, err, tc.Error)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, tokens)
			}
		})
	}
}

func BenchmarkEncodingXMLDecoder(b *testing.B) {
	data := benchData(b)
	for n := 0; n < b.N; n++ {
		d := xml.NewDecoder(bytes.NewReader(data))
		for {
			_, err := d.RawToken()
			if err == io.EOF {
				break
			} else if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	}
}

func BenchmarkXMLTokenReader(b *testing.B) {
	data := benchData(b)
	for n := 0; n < b.N; n++ {
		d := NewXMLTokenReader(NewScanner(data))
		for {
			_, err := d.Token()
			if err == io.EOF {
				break
			} else if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	}
}
