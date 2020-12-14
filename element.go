package fastxml

import (
	"bytes"
	"errors"
)

// Allocate the errors once and return the same structs
var (
	errAttrPrefix = errors.New(`expected Attr to start with '"'`)
	errAttrSuffix = errors.New(`expected Attr to end with '"'`)
	errWhitespace = errors.New(`expected whitespace but got non-whitespace`)
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
	// If there are attributes present
	if space := bytes.IndexByte(token[start:end], ' '); space != -1 {
		return token[start : start+space], token[space+start+1 : end]
	}
	// handle self-closing elements
	if token[end-1] == '/' {
		end--
	}
	// No attributes
	return token[start:end], nil
}

// Attrs calls f for each key="value" in token, stopping if f returns true
// The value will _not_ be decoded yet
func Attrs(token []byte, f func(key []byte, value []byte) error) error {
	offset := 0
	for offset < len(token) {
		// Find the next `=` section
		equal := bytes.IndexByte(token[offset:], '=')
		if equal == -1 {
			break // End of element
		}
		// Extract the key
		key := bytes.TrimSpace(token[offset : offset+equal])
		offset += equal + 1
		// Find the `"` to start the value
		start := bytes.IndexByte(token[offset:], '"')
		if start == -1 {
			return errAttrPrefix
		}
		offset += start + 1
		// Find the `"` to end the value
		end := bytes.IndexByte(token[offset:], '"')
		if end == -1 {
			return errAttrSuffix
		}
		value := token[offset : offset+end]
		offset += end + 1
		// Trigger the callback
		if err := f(key, value); err != nil {
			return err
		}
	}
	// Make sure no extra values in
	if len(bytes.TrimSpace(token[offset:])) > 0 {
		return errWhitespace
	}
	return nil
}
