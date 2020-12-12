package fastxml

import "encoding/xml"

func makeCopy(b []byte) []byte {
	b1 := make([]byte, len(b))
	copy(b1, b)
	return b1
}

// A Token is an interface holding one of the token types: StartElement, EndElement, CDATA, CharData, Comment, ProcInst, or Directive.
type Token interface{}

// A Name represents an XML name (Local) annotated with a name space identifier (Space).
// For fastxml the Space identifier is given as the short prefix used in the document being parsed, not the canonical URL.
type Name struct {
	Space []byte
	Local []byte
}

// Copy creates a new copy of Name.
func (n Name) Copy() Name {
	n.Space = makeCopy(n.Space)
	n.Local = makeCopy(n.Local)
	return n
}

// XML converts to a xml.Name (using unsafe strings)
func (n *Name) XML() xml.Name {
	return xml.Name{
		Space: unsafeString(n.Space),
		Local: unsafeString(n.Local),
	}
}

// An Attr represents an attribute in an XML element (Name=Value).
type Attr struct {
	Name Name
	// Value should be passed through DecodeEntities before usage
	Value []byte
}

// Copy creates a new copy of Attr.
func (a Attr) Copy() Attr {
	a.Name = a.Name.Copy()
	a.Value = makeCopy(a.Value)
	return a
}

// XML converts to a xml.Attr
func (a *Attr) XML() (xml.Attr, error) {
	decoded, err := DecodeEntities(a.Value)
	if err != nil {
		return xml.Attr{}, err
	}
	return xml.Attr{
		Name:  a.Name.XML(),
		Value: unsafeString(decoded),
	}, nil
}

// A StartElement represents an XML start element.
type StartElement struct {
	Name Name
	Attr []Attr
}

// End returns the matching EndElement for the StartElement
func (e StartElement) End() EndElement {
	return EndElement{Name: e.Name}
}

// Copy creates a new copy of StartElement.
func (e StartElement) Copy() StartElement {
	e.Name = e.Name.Copy()
	attrs := make([]Attr, len(e.Attr))
	for idx, attr := range e.Attr {
		attrs[idx] = attr.Copy()
	}
	e.Attr = attrs
	return e
}

// XML converts a xml.StartElement
func (e *StartElement) XML() (xml.StartElement, error) {
	se := xml.StartElement{
		Name: e.Name.XML(),
		Attr: make([]xml.Attr, len(e.Attr)),
	}
	for idx, attr := range e.Attr {
		var err error
		se.Attr[idx], err = attr.XML()
		if err != nil {
			return xml.StartElement{}, err
		}
	}
	return se, nil
}

// Token converts a xml.Token
func (e *StartElement) Token() (xml.Token, error) {
	return e.XML()
}

// An EndElement represents an XML end element.
type EndElement struct {
	Name Name
}

// Copy creates a new copy of EndElement.
func (e EndElement) Copy() EndElement {
	e.Name = e.Name.Copy()
	return e
}

// XML converts a xml.EndElement
func (e *EndElement) XML() (xml.EndElement, error) {
	return xml.EndElement{
		Name: e.Name.XML(),
	}, nil
}

// Token converts a xml.Token
func (e *EndElement) Token() (xml.Token, error) {
	return e.XML()
}

// A CDATA represents XML character data (raw text) from a <[CDATA[...]]> section
type CDATA []byte

// Copy creates a new copy of CDATA.
func (c CDATA) Copy() CDATA {
	return CDATA(makeCopy(c))
}

// XML converts a xml.CharData
func (c CDATA) XML() (xml.CharData, error) {
	return xml.CharData([]byte(c)), nil
}

// Token converts a xml.Token
func (c *CDATA) Token() (xml.Token, error) {
	return c.XML()
}

// A CharData represents XML character data (raw text).
// XML escape sequences have NOT been replaced by the characters they represent.
type CharData []byte

// Copy creates a new copy of CharData.
func (c CharData) Copy() CharData {
	return CharData(makeCopy(c))
}

// XML converts a xml.CharData
func (c CharData) XML() (xml.CharData, error) {
	decoded, err := DecodeEntities([]byte(c))
	return xml.CharData(decoded), err
}

// Token converts a xml.Token
func (c *CharData) Token() (xml.Token, error) {
	return c.XML()
}

// A Comment represents an XML comment of the form <!--comment-->. The bytes do not include the <!-- and --> comment markers.
type Comment []byte

// Copy creates a new copy of Comment.
func (c Comment) Copy() Comment {
	return Comment(makeCopy(c))
}

// XML converts a xml.Comment
func (c Comment) XML() (xml.Comment, error) {
	decoded, err := DecodeEntities([]byte(c))
	return xml.Comment(decoded), err
}

// Token converts a xml.Token
func (c Comment) Token() (xml.Token, error) {
	return c.XML()
}

// A ProcInst represents an XML processing instruction of the form <?target inst?>
type ProcInst struct {
	Target []byte
	Inst   []byte
}

// Copy creates a new copy of CharData.
func (p ProcInst) Copy() ProcInst {
	p.Target = makeCopy(p.Target)
	p.Inst = makeCopy(p.Inst)
	return p
}

// XML converts a xml.ProcInst
func (p *ProcInst) XML() (xml.ProcInst, error) {
	return xml.ProcInst{
		Target: unsafeString(p.Target),
		Inst:   p.Inst,
	}, nil
}

// Token converts a xml.Token
func (p *ProcInst) Token() (xml.Token, error) {
	return p.XML()
}

// A Directive represents an XML directive of the form <!text>. The bytes do not include the <! and > markers.
type Directive []byte

// Copy creates a new copy of Directive.
func (d Directive) Copy() Directive {
	return Directive(makeCopy(d))
}

// XML converts a xml.Comment
func (d Directive) XML() (xml.Directive, error) {
	return xml.Directive([]byte(d)), nil
}

// Token converts a xml.Token
func (d Directive) Token() (xml.Token, error) {
	return d.XML()
}

// CopyToken returns a copy of a Token.
func CopyToken(t Token) Token {
	switch v := t.(type) {
	case CharData:
		return v.Copy()
	case CDATA:
		return v.Copy()
	case Comment:
		return v.Copy()
	case Directive:
		return v.Copy()
	case ProcInst:
		return v.Copy()
	case StartElement:
		return v.Copy()
	}
	return t
}
