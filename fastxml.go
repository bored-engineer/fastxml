package fastxml

import (
	"bytes"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"strconv"
	"unicode"
	"unsafe"
)

// entities is xml.HTMLEntity but with lt/gt/amp/apos/quot added
var entities = make(map[string]string)

func init() {
	// Copy in everything from xml.HTMLEntity
	for k, v := range xml.HTMLEntity {
		entities[k] = v
	}
	// These are hardcoded in (encoding/xml).Decoder.Entity
	entities["lt"] = "<"
	entities["gt"] = ">"
	entities["amp"] = "&"
	entities["apos"] = "'"
	entities["quot"] = `"`
}

type TokenReader struct {
	// buf is the raw byte slice we are parsing
	// It is and MUST be immutable
	buf []byte
	// cursor is the offset in buf we are currently at
	cursor int
	// length is the size of buf
	length int
	// nextToken is used when there is a self-terminated element
	// if populated the next call to Token returns it
	nextToken *xml.EndElement
}

// unsafeString performs an _unsafe_ no-copy string allocation from bs
// https://github.com/golang/go/issues/25484 has more info on this
// the implementation is roughly taken from strings.Builder's
func unsafeString(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

// string calls unsafeString on tr.buf from start to end
func (tr *TokenReader) string(start, end int) string {
	return unsafeString(tr.buf[start:end])
}

// indexRuneWithin find the next rune within bounds of end
func (tr *TokenReader) indexRuneWithin(r rune, end int) int {
	idx := bytes.IndexRune(tr.buf[tr.cursor:end], r)
	if idx != -1 {
		return idx + tr.cursor
	}
	return idx
}

// indexRune find the next instance of r in buf starting at cursor
func (tr *TokenReader) indexRune(r rune) int {
	idx := bytes.IndexRune(tr.buf[tr.cursor:], r)
	if idx != -1 {
		return idx + tr.cursor
	}
	return idx
}

// indexString find the next instance of needle in buf starting at cursor
func (tr *TokenReader) indexString(needle string) int {
	idx := bytes.Index(tr.buf[tr.cursor:], []byte(needle))
	if idx != -1 {
		return idx + tr.cursor
	}
	return idx
}

// Token returns the next in b
func (tr *TokenReader) Token() (xml.Token, error) {
	// If we already have a pending token, return that and clean it up
	if tr.nextToken != nil {
		token := *tr.nextToken
		tr.nextToken = nil
		return token, nil
	}
	// If we are at the end of buf, stop parsing
	if tr.cursor >= tr.length {
		return nil, nil
	}
	// If it doesn't start with a <, it's CharData
	if tr.buf[tr.cursor] != '<' {
		// Must be a CharData token
		return tr.parseCharData()
	}
	// Move cursor past '<'
	tr.cursor += 1
	// Make sure we have enough characters to make a valid element
	// Smallest element will be <a>
	if rem := tr.length - tr.cursor; rem < 2 {
		return nil, fmt.Errorf(
			"Not enough bytes (%d) remaining for valid XML element declaration",
			rem,
		)
	}
	// Check if the next byte is a comment or declaration
	// This is safe due to above length check
	switch tr.buf[tr.cursor] {
	// ProcInst
	case '?':
		// Move cursor beyond ?
		tr.cursor += 1
		return tr.parseProcInst()
	case '!':
		// Move cursor beyond !
		tr.cursor += 1
		return tr.parsePotentialDirective()
	default:
		return tr.parseElement()
	}
}

// indexError generates useful errors when indexes fail
func (tr *TokenReader) indexError(needle string) error {
	return fmt.Errorf(
		"Couldn't find XML %s in: %v",
		needle,
		unsafeString(tr.buf[tr.cursor:]),
	)
}

// skipSpace finds the first non-space value
// TODO: This could walk past end
func (tr *TokenReader) skipSpace(start int) int {
	for unicode.IsSpace(rune(tr.buf[start])) {
		start += 1
	}
	return start
}

// reverseSpace trims the last non-space value
// TODO: This could walk past start
func (tr *TokenReader) reverseSpace(end int) int {
	for unicode.IsSpace(rune(tr.buf[end])) {
		end -= 1
	}
	return end + 1
}

// decode converts any entities to their matched value
// TODO: This probably panics with invalid entities, make safe
func (tr *TokenReader) decode(stopIdx int) ([]byte, error) {
	// Save the original cursor location
	startIdx := tr.indexRuneWithin('&', stopIdx)
	// If there are no entities, don't do an expensive compare
	if startIdx == -1 {
		return tr.buf[tr.cursor:stopIdx], nil
	}
	// Start a new byte slice that has the length of the decoded bytes
	// all entities are smaller than their name (ex: &quot; becomes ")
	// if this is not the case, this function breaks
	// if we ever add support for custom entities, will need to refactor
	result := make([]byte, stopIdx-tr.cursor)
	size := 0
	// Loop until we find no more entities
	for {
		// Copy in the bytes up to the entity as-is
		size += copy(result[size:], tr.buf[tr.cursor:startIdx])
		tr.cursor = startIdx + 1
		// Find the end of the entity
		endIdx := tr.indexRuneWithin(';', stopIdx)
		// If there is no element end, skip over this byte
		if endIdx == -1 {
			return nil, tr.indexError("Entity end")
		}
		// If the element is a rune by hex/decimal name
		if tr.buf[tr.cursor] == '#' {
			tr.cursor += 1
			// hex vs decimal
			if tr.buf[tr.cursor] == 'x' {
				tr.cursor += 1
				// Decode directly into the result slice, returning errs
				added, err := hex.Decode(
					result[size:],
					tr.buf[tr.cursor:endIdx],
				)
				if err != nil {
					return nil, err
				}
				size += added
			} else {
				// Use unsafe to get a string for strconv
				// See also https://github.com/golang/go/issues/2632
				numStr := tr.string(tr.cursor, endIdx)
				num, err := strconv.Atoi(numStr)
				if err != nil {
					return nil, fmt.Errorf(
						"Invalid XML decimal entity: %v",
						err,
					)
				}
				result[size] = byte(rune(num))
				size += 1
			}
		} else {
			// Must be a named entity, calculate the name
			name := tr.string(tr.cursor, endIdx)
			// Get the entity by name from the internal map
			// TODO: Is a massive switch faster?
			sub, ok := entities[name]
			if !ok {
				return nil, fmt.Errorf("Unknown XML entity: %v", name)
			}
			// Copy in the replaced entity
			size += copy(result[size:], sub)
		}
		// Reset cursor past the end of this entity
		tr.cursor = endIdx + 1
		// Then search for the next entity
		startIdx = tr.indexRuneWithin('&', stopIdx)
		// If no next entity, bail
		if startIdx == -1 {
			break
		}
	}
	// Copy in the rest of the data and return
	size += copy(result[size:], tr.buf[tr.cursor:stopIdx])
	tr.cursor = stopIdx
	return result[0:size], nil
}

// parseName parses a xml.Name from a byte slice
func (tr *TokenReader) parseName(start, end int) xml.Name {
	start = tr.skipSpace(start)
	end = tr.reverseSpace(end - 1)
	colonIdx := bytes.IndexRune(tr.buf[start:end], ':')
	if colonIdx != -1 {
		return xml.Name{
			Local: tr.string(colonIdx+1+start, end),
			Space: tr.string(start, colonIdx+start),
		}
	}
	return xml.Name{
		Local: tr.string(start, end),
	}
}

// parseAttrs parses the attributes within an element
func (tr *TokenReader) parseAttrs(stopIdx int) ([]xml.Attr, error) {
	var attrs []xml.Attr
	for {
		// Find the location of the =
		equalIdx := tr.indexRuneWithin('=', stopIdx)
		// If none found, end of attributes, bail
		if equalIdx == -1 {
			break
		}
		name := tr.parseName(tr.cursor, equalIdx)
		tr.cursor = equalIdx + 1
		// Search for the start of the attribute value
		if tr.buf[tr.cursor] == '"' {
			// Move cursor past the value
			tr.cursor += 1
		} else {
			// Move cursor past the start of the quote
			startIdx := tr.indexRuneWithin('"', stopIdx)
			if startIdx == -1 {
				return nil, tr.indexError("Attribute start quote")
			}
			tr.cursor = startIdx + 1
		}
		// Find the end of the attribute value
		endIdx := tr.indexRuneWithin('"', stopIdx)
		if endIdx == -1 {
			return nil, tr.indexError("Attribute end quote")
		}
		value, err := tr.decode(endIdx)
		if err != nil {
			return nil, err
		}
		// Add it to attributes and adjust cursor past end quote
		tr.cursor = endIdx + 1
		attrs = append(attrs, xml.Attr{
			Name:  name,
			Value: unsafeString(value),
		})
	}
	return attrs, nil
}

// parseElement parses a xml.Element
func (tr *TokenReader) parseElement() (xml.Token, error) {
	// Find the end of the element
	endIdx := tr.indexRune('>')
	if endIdx == -1 {
		return nil, tr.indexError("Element end")
	}
	// selfClosingElement are ones that end with '/'
	selfClosingElement := (tr.buf[endIdx-1] == '/')
	// endElement are ones that start with '/'
	endElement := (tr.buf[tr.cursor] == '/')
	// By default the nameIdx is just the end of the element
	var nameIdx int
	if endElement {
		// Skip the '/' when parsing name
		tr.cursor += 1
		nameIdx = endIdx
	} else {
		nameIdx = tr.indexRuneWithin(' ', endIdx)
		if nameIdx == -1 {
			if selfClosingElement {
				// Skip the final '/' when parsing name
				nameIdx = endIdx - 1
			} else {
				nameIdx = endIdx
			}
		}
	}
	name := tr.parseName(tr.cursor, nameIdx)
	// If it's an end element, bail here early
	if endElement {
		return &xml.EndElement{
			Name: name,
		}, nil
	}
	// If it ends with / it's an self closing element, add a nextToken
	if selfClosingElement {
		tr.nextToken = &xml.EndElement{
			Name: name,
		}
	}
	// If there are no attributes, fast-path return
	if nameIdx == endIdx || (selfClosingElement && nameIdx == endIdx-1) {
		// Adjust cursor and return
		tr.cursor = endIdx + 1
		return &xml.StartElement{
			Name: name,
			Attr: nil,
		}, nil
	}
	// Must be attrs to reach this point, parse them
	tr.cursor = nameIdx + 1
	attrs, err := tr.parseAttrs(endIdx)
	if err != nil {
		return nil, err
	}
	// Adjust cursor and return
	tr.cursor = endIdx + 1
	return &xml.StartElement{
		Name: name,
		Attr: attrs,
	}, nil
}

// parseCharData parses xml.CharData from buf
func (tr *TokenReader) parseProcInst() (xml.Token, error) {
	// ProcInst have a target and data following that
	targetIdx := tr.indexRune(' ')
	if targetIdx == -1 {
		return nil, tr.indexError("ProcInst target")
	}
	target := tr.string(tr.cursor, targetIdx)
	tr.cursor = targetIdx + 1
	// Find the end of the comment, everything in between is the data
	endIdx := tr.indexString("?>")
	if endIdx == -1 {
		return nil, tr.indexError("ProcInst end")
	}
	data := tr.buf[tr.cursor:endIdx]
	// Adjust cursor and return the data
	tr.cursor = endIdx + 2
	return xml.ProcInst{
		Target: target,
		Inst:   data,
	}, nil
}

// parseComment parses a xml.Comment
func (tr *TokenReader) parseComment() (xml.Comment, error) {
	// Find the end of the comment
	endIdx := tr.indexString("-->")
	if endIdx == -1 {
		return nil, tr.indexError("Comment end")
	}
	data := tr.buf[tr.cursor:endIdx]
	// Adjust cursor and return
	tr.cursor = endIdx + 3
	return xml.Comment(data), nil
}

// parseCDATA parses a CDATA section (CharData)
func (tr *TokenReader) parseCDATA() (xml.CharData, error) {
	// Find the end of the CDATA
	endIdx := tr.indexString("]]>")
	if endIdx == -1 {
		return nil, tr.indexError("CDATA end")
	}
	// NOTE: No decoding needed for CDATA
	data := tr.buf[tr.cursor:endIdx]
	// Adjust cursor and return
	tr.cursor = endIdx + 3
	return xml.CharData(data), nil
}

// parsePotentialDirective parses potential xml.Directive elements
// but also Comment and CharData via CDATA can be returned
func (tr *TokenReader) parsePotentialDirective() (xml.Token, error) {
	switch tr.buf[tr.cursor] {
	// Potential comment
	case '-':
		// Make sure long enough for a full comment start '-'
		rem := tr.length - tr.cursor
		// If <!--, parse as a comment
		if rem >= 1 && tr.buf[tr.cursor+1] == '-' {
			tr.cursor += 1
			return tr.parseComment()
		}
	// Potential CDATA
	case '[':
		// Make sure long enough for a full cdata '[CDATA['
		rem := tr.length - tr.cursor
		// If <!--, parse as a comment
		if rem >= 7 && bytes.Equal(tr.buf[tr.cursor+1:tr.cursor+8], []byte("[CDATA[")) {
			tr.cursor += 7
			return tr.parseCDATA()
		}
	}
	// Must be an actual directive, find the end of it, middle is data
	endIdx := tr.indexRune('>')
	data := tr.buf[tr.cursor:endIdx]
	// Adjust cursor and return
	tr.cursor = endIdx + 1
	return xml.Directive(data), nil
}

// parseCharData parses xml.CharData from buf
func (tr *TokenReader) parseCharData() (xml.Token, error) {
	// CharData ends at the next element start
	end := tr.indexRune('<')
	// If there is no next element, use end of buf as end
	if end == -1 {
		end = tr.length
	}
	// CharData can contain entities, decode them
	decoded, err := tr.decode(end)
	if err != nil {
		return nil, err
	}
	// Adjust cursor and return
	tr.cursor = end
	return xml.CharData(decoded), nil
}

// NewTokenReader creates a *TokenReader instance given a byte slice.
// It is critical that bs is not modified after it is passed to TokenReader
func NewTokenReader(bs []byte) *TokenReader {
	return &TokenReader{
		buf:    bs,
		cursor: 0,
		// calculate once for speed
		length: len(bs),
	}
}
