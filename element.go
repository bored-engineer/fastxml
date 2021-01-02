package fastxml

import (
	"bytes"
	"errors"
	"fmt"
	"unicode"
)

// Allocate the errors once and return the same structs
var (
	errAttrKeyWhitespace = errors.New(`expected Attr to have a non-whitespace key`)
	errAttrPrefix        = errors.New(`expected Attr to start with '"'`)
	errAttrSuffix        = errors.New(`expected Attr to end with '"'`)
)

// IsElement checks if a []byte is an element (is not a ProcInst or Directive)
func IsElement(token []byte) bool {
	// Not Directive or ProcInst
	return len(token) >= 3 && token[0] == '<' && token[1] != '!' && token[1] != '?'
}

// IsSelfClosing checks if a []byte is an self closing element (<element/>)
func IsSelfClosing(token []byte) bool {
	if len(token) <= 2 {
		return false
	}
	return token[len(token)-2] == '/'
}

// IsEndElement checks if a []byte is a </element>
func IsEndElement(token []byte) bool {
	return len(token) >= 2 && token[0] == '<' && token[1] == '/'
}

// IsStartElement is the inverse of IsEndElement
func IsStartElement(token []byte) bool {
	return len(token) >= 2 && token[0] == '<' && token[1] != '/'
}

// Element extracts the name of the element (ex: `<foo:bar key="val"/>` -> `foo:bar`) and attribute sections
func Element(token []byte) (name []byte, attrs []byte) {
	if len(token) < 3 {
		return nil, nil
	}
	// Find the start and end of the element
	end := len(token) - 1
	start := 1
	if token[start] == '/' {
		start++ // handle end elements
	}
	// handle self-closing elements
	if token[end-1] == '/' {
		end--
	}
	// If there are attributes present
	if space := bytes.IndexByte(token[start:end], ' '); space != -1 {
		return token[start : start+space], token[space+start+1 : end]
	}
	// No attributes
	return token[start:end], nil
}

// notSpace is the inverse of unicode.IsSpace
func notSpace(r rune) bool {
	return !unicode.IsSpace(r)
}

// RawAttrs calls f for each key="value" in token, stopping if f returns false
func RawAttrs(attrsToken []byte, f func(keyStart, keyEnd, valueStart, valueEnd int) bool) error {
	offset := 0
	for offset < len(attrsToken) {
		// Find the next `=` section:
		equals := bytes.IndexByte(attrsToken[offset:], '=')
		if equals == -1 {
			break // End of element
		}
		equals += offset
		// ` key = "value"`
		//       ^
		// Extract the key offsets
		keyStart := offset
		// Trim any whitespace on the key name
		if idx := bytes.IndexFunc(attrsToken[offset:equals], notSpace); idx == -1 {
			return errAttrKeyWhitespace
		} else if idx > 0 {
			keyStart += idx
		}
		// Don't need to check for -1 here as IndexFunc would have found it
		keyEnd := keyStart
		if idx := bytes.LastIndexFunc(attrsToken[keyStart:equals], notSpace); idx > 0 {
			keyEnd += idx + 1
		}
		// Move past the end of the equals statement
		// ` key = "value"`
		//        ^
		equals++
		// Find the `"` to start the value
		valueStart := bytes.IndexByte(attrsToken[equals:], '"')
		if valueStart == -1 {
			return errAttrPrefix
		}
		// ` key = "value"`
		//          ^
		valueStart += equals + 1
		// Find the `"` to end the value
		// ` key = "value"`
		//               ^
		valueEnd := bytes.IndexByte(attrsToken[valueStart:], '"')
		if valueEnd == -1 {
			return errAttrSuffix
		}
		valueEnd += valueStart
		// Move to end of value
		// ` key = "value"`
		//                ^
		offset = valueEnd + 1
		// Trigger the callback stopping iteration as needed
		if !f(keyStart, keyEnd, valueStart, valueEnd) {
			return nil
		}
	}
	// Make sure no extra values in
	if idx := bytes.IndexFunc(attrsToken[offset:], notSpace); idx != -1 {
		return fmt.Errorf("expected whitespace but got %q", String(attrsToken[offset+idx:]))
	}
	return nil
}

// Attrs calls f for each key="value" in token, stopping if f returns false
// The value will _not_ be decoded yet
func Attrs(attrsToken []byte, f func(key []byte, value []byte) bool) error {
	return RawAttrs(attrsToken, func(keyStart, keyEnd, valueStart, valueEnd int) bool {
		return f(attrsToken[keyStart:keyEnd], attrsToken[valueStart:valueEnd])
	})
}

// RawAttr reads a specific attribute value (or -1 if not found)
func RawAttr(attrsToken []byte, attrKey []byte) (start int, stop int, err error) {
	start, stop = -1, -1
	err = RawAttrs(attrsToken, func(keyStart, keyStop, valueStart, valueStop int) bool {
		if bytes.Equal(attrsToken[keyStart:keyStop], attrKey) {
			start, stop = valueStart, valueStop
			return false
		}
		return true
	})
	return
}

// Attr reads a specific attribute and returns the (non-decoded) value
func Attr(attrsToken []byte, attrKey []byte) (attrValue []byte, err error) {
	start, stop, err := RawAttr(attrsToken, attrKey)
	if err != nil {
		return nil, err
	} else if start == -1 {
		return nil, nil
	}
	return attrsToken[start:stop], nil
}
