package fastxml

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

var (
	prefixComment  = []byte("--")
	prefixCDATA    = []byte("[CDATA[")
	suffixCDATA    = []byte("]]>")
	suffixProcInst = []byte("?>")
	suffixComment  = []byte("-->")
)

// parseProcInst parses a "<?target inst?>"
func parseProcInst(buf []byte) (Token, int, error) {
	// Find the first space
	space := bytes.IndexByte(buf, ' ')
	if space == -1 {
		return nil, 0, errors.New("expected ' ' in ProcInst")
	}
	// Find the end of the ProcInst
	end := bytes.Index(buf[space:], suffixProcInst)
	if end == -1 {
		return buf, 0, errors.New("expected '?>' to end ProcInst")
	}
	end += space
	// Return the ProcInst
	return ProcInst{
		Target: buf[:space],
		Inst:   buf[space+1 : end],
	}, end + 2, nil
}

// parsePotentialDirective parses a <!directive> or <![CDATA[]]> or <!--comment-->
func parsePotentialDirective(buf []byte) (Token, int, error) {
	if bytes.HasPrefix(buf, prefixCDATA) {
		// Find the end of the CDATA
		end := bytes.Index(buf, suffixCDATA)
		if end == -1 {
			return nil, 0, errors.New("expected ']]>' to end CDATA")
		}
		// Return the CDATA
		return CDATA(buf[7:end]), end + 3, nil
	} else if bytes.HasPrefix(buf, prefixComment) {
		// Find the end of the Comment
		end := bytes.Index(buf, suffixComment)
		if end == -1 {
			return nil, 0, errors.New("expected '-->' to end Comment")
		}
		return Comment(buf[2:end]), end + 3, nil
	}
	// Find the end of the Directive
	end := bytes.IndexByte(buf, '>')
	if end == -1 {
		return nil, 0, errors.New("expected '>' to end Directive")
	}
	// Return the Directive
	return Directive(buf[:end]), end + 1, nil
}

// parseName parses a Name
func parseName(buf []byte) Name {
	if idx := bytes.IndexByte(buf, ':'); idx != -1 {
		return Name{
			Space: buf[:idx],
			Local: buf[idx+1:],
		}
	}
	return Name{Local: buf}
}

// parseElement parses a <element key="value">
func parseElement(buf []byte) (Token, int, bool, error) {
	// Find the end of the element
	end := bytes.IndexByte(buf, '>')
	if end == -1 {
		return nil, 0, false, errors.New("expected '>' to end StartElement")
	}
	offset := end + 1
	// Self closing element
	closing := (buf[end-1] == '/')
	if closing {
		end--
	}
	// Find the first ' ' after the element name if present
	space := bytes.IndexByte(buf[:end], ' ')
	if space == -1 {
		// Element with no attributes
		return StartElement{
			Name: parseName(buf[:end]),
		}, offset, closing, nil
	}
	// Element with attributes
	se := StartElement{
		Name: parseName(buf[:space]),
	}
	// Loop each attribute
	cursor := space + 1
	for {
		// Find the next attribute
		equal := bytes.IndexByte(buf[cursor:end], '=')
		if equal == -1 {
			// No more attributes
			break
		}
		name := parseName(bytes.TrimSpace(buf[cursor : cursor+equal]))
		cursor += equal
		// Find the value start
		quoteStart := bytes.IndexByte(buf[cursor:end], '"')
		if quoteStart == -1 {
			return nil, 0, false, errors.New(`expected '"' to start Attr`)
		}
		cursor += quoteStart + 1
		// Find the value end
		quoteEnd := bytes.IndexByte(buf[cursor:end], '"')
		if quoteEnd == -1 {
			return nil, 0, false, errors.New(`expected '"' to end Attr`)
		}
		// Add the attribute
		se.Attr = append(se.Attr, Attr{
			Name:  name,
			Value: buf[cursor : cursor+quoteEnd],
		})
		cursor += quoteEnd + 1
	}
	// Check to make sure no weird extra values
	for ; cursor < end; cursor++ {
		if buf[cursor] != ' ' {
			return nil, 0, false, fmt.Errorf("unexpected %q after Attrs", string(buf[cursor]))
		}
	}
	// Return the StartElement
	return se, offset, closing, nil
}

// Decoder reads Tokens from a []byte
type Decoder struct {
	// buf is the raw byte slice we are parsing
	// It is and MUST be immutable
	buf []byte
	// cursor is the offset in buf we are currently at
	cursor int
	// length is the size of buf
	length int
	// next is the next element start (`<`)
	next int
	// nextToken is used when t here is a self-terminated element
	// if populated the next call to Token returns it
	nextToken *EndElement
}

// InputOffset returns the offset the reader is at
func (d *Decoder) InputOffset() int64 {
	return int64(d.cursor)
}

// RawToken returns the next token value
func (d *Decoder) RawToken() (Token, error) {
	// If we already have a pending token, return that and clean it up
	if d.nextToken != nil {
		token := *d.nextToken
		d.nextToken = nil
		return token, nil
	}
	// If cursor at end of buffer, it's the end of the file
	if d.cursor >= len(d.buf) {
		return nil, io.EOF
	}
	// If there is gap between the cursor and the next '<', we have some chardata to emit
	if d.cursor < d.next {
		token := CharData(d.buf[d.cursor:d.next])
		d.cursor = d.next
		return token, nil
	}
	// Cursor is at '<'
	d.cursor++
	// Make sure we have enough characters to make a valid element
	// Smallest element will be <a>
	if rem := len(d.buf) - d.cursor; rem < 2 {
		return nil, fmt.Errorf("not enough bytes (%d) remaining for valid XML element declaration", rem)
	}
	// Check if the next byte is a comment or declaration
	// This is safe due to above length check
	var offset int
	var err error
	var token Token
	switch d.buf[d.cursor] {
	// ProcInst
	case '?':
		// Move cursor beyond ?
		d.cursor++
		token, offset, err = parseProcInst(d.buf[d.cursor:])
	// Directive
	case '!':
		// Move cursor beyond !
		d.cursor++
		token, offset, err = parsePotentialDirective(d.buf[d.cursor:])
	// Must be an Element
	default:
		var closing bool
		token, offset, closing, err = parseElement(d.buf[d.cursor:])
		if closing {
			// Self-closing, setup the next element when token is called
			end := token.(StartElement).End()
			d.nextToken = &end
		}
	}
	d.cursor += offset
	if err != nil {
		return nil, err
	}
	// Find the next token pre-emptively
	if next := bytes.IndexByte(d.buf[d.cursor:], '<'); next == -1 {
		d.next = len(d.buf)
	} else {
		d.next = d.cursor + next
	}
	return token, nil
}

// NewDecoder creates a *Decoder instance given a byte slice.
// It is critical that bs is not modified after it is passed to Decoder
func NewDecoder(bs []byte) *Decoder {
	return &Decoder{
		buf:    bs,
		cursor: 0,
		next:   bytes.IndexByte(bs, '<'),
	}
}
