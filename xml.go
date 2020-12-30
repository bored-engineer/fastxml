package fastxml

import (
	"encoding/xml"
	"fmt"
	"sync"
)

// XMLCharData produces a xml.CharData given a token
func XMLCharData(token []byte) (xml.CharData, error) {
	cd, err := CharData(token, nil)
	if err != nil {
		return nil, err
	}
	return xml.CharData(cd), nil
}

// XMLDirective produces a xml.Directive given a token
func XMLDirective(token []byte) xml.Directive {
	return xml.Directive(Directive(token))
}

// XMLComment produces a xml.Comment given a token
func XMLComment(token []byte) xml.Comment {
	return xml.Comment(Comment(token))
}

// XMLProcInst produces a xml.ProcInst given a token
func XMLProcInst(token []byte) xml.ProcInst {
	target, inst := ProcInst(token)
	return xml.ProcInst{
		Target: String(target),
		Inst:   inst,
	}
}

// XMLName produces a xml.Name given a token
func XMLName(token []byte) xml.Name {
	space, local := Name(token)
	return xml.Name{
		Space: String(space),
		Local: String(local),
	}
}

// XMLAttr produces a xml.Attr given a key, value
func XMLAttr(key []byte, value []byte) (attr xml.Attr, err error) {
	value, err = DecodeEntities(value, nil)
	if err != nil {
		return
	}
	attr.Name = XMLName(key)
	attr.Value = String(value)
	return
}

// reduce allocations when casting many attributes
var attrsPool = &sync.Pool{
	New: func() interface{} {
		// pre-allocate a few elements to avoid repeated growth of slices
		return make([]xml.Attr, 0, 3)
	},
}

// XMLAttrs produces a []xml.Attr given attributes slice
func XMLAttrs(token []byte) ([]xml.Attr, error) {
	attrs := attrsPool.Get().([]xml.Attr)
	// Loop each attribute
	if err := Attrs(token, func(key []byte, value []byte) error {
		attr, err := XMLAttr(key, value)
		if err != nil {
			return err
		}
		attrs = append(attrs, attr)
		return nil
	}); err != nil {
		return nil, err
	}
	// If no attributes
	if len(attrs) == 0 {
		attrsPool.Put(attrs)
		// Use nil so gc can cleanup attrs slice
		return nil, nil
	}
	return attrs, nil
}

// XMLStartElement produces a xml.StartElement given a token
func XMLStartElement(token []byte) (xml.StartElement, error) {
	name, attrToken := Element(token)
	attrs, err := XMLAttrs(attrToken)
	if err != nil {
		return xml.StartElement{}, err
	}
	return xml.StartElement{
		Name: XMLName(name),
		Attr: attrs,
	}, nil
}

// XMLEndElement produces a xml.EndElement given a token
func XMLEndElement(token []byte) xml.EndElement {
	name, _ := Element(token)
	return xml.EndElement{
		Name: XMLName(name),
	}
}

// XMLElement produces a xml.EndElement or xml.StartElement depending on IsEndElement
func XMLElement(token []byte) (xml.Token, error) {
	if IsEndElement(token) {
		return XMLEndElement(token), nil
	}
	return XMLStartElement(token)
}

// XMLToken produces a xml.Token given a piece of data
func XMLToken(token []byte, chardata bool) (xml.Token, error) {
	switch {
	case chardata:
		return XMLCharData(token)
	case IsDirective(token):
		return XMLDirective(token), nil
	case IsComment(token):
		return XMLComment(token), nil
	case IsProcInst(token):
		return XMLProcInst(token), nil
	default:
		return XMLElement(token)
	}
}

// tokenReader implements xml.TokenReader given a *Scanner
type tokenReader struct {
	s    *Scanner
	next *xml.EndElement
}

// Token implements xml.TokenReader
func (tr *tokenReader) Token() (_ xml.Token, err error) {
	// Just in case that data was not well-formed or some other error
	defer func() {
		if rErr := recover(); rErr != nil {
			err = fmt.Errorf("unexpected panic: %v", rErr)
		}
	}()
	// If we have a next token use that
	if tr.next != nil {
		token := *tr.next
		tr.next = nil
		return token, nil
	}
	// Get the next token, convert to XML interface
	rawToken, chardata, sErr := tr.s.Next()
	if sErr != nil {
		return nil, sErr
	}
	token, tErr := XMLToken(rawToken, chardata)
	if tErr != nil {
		return nil, tErr
	}
	// If it was a element and it's self closing, next token is it's end element
	if start, ok := token.(xml.StartElement); ok && IsSelfClosing(rawToken) {
		end := start.End()
		tr.next = &end
	}
	return token, nil
}

// NewXMLTokenReader creates a xml.TokenReader given a scanner
func NewXMLTokenReader(s *Scanner) xml.TokenReader {
	return &tokenReader{s: s}
}
