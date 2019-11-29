package main

import (
	"fmt"
	"unsafe"
	"bytes"
	"strings"
	"strconv"
	"encoding/xml"
)

// unsafeString performs a no-copy cast of a byte slice to a string
// https://github.com/golang/go/issues/25484 has some details on this
// this is fast but also has the potential to create mutable strings
// if we assume the b is immutable this is "ok"
// this is the same method used by strings.Builder
func unsafeString(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

// decodedValue replaces entities with their real value as fast as possible
// TODO: some cases do a string([]byte(string(val))) when calling this method
func decodedValue(b []byte) []byte {
	// Get the location of the first entity start
	beginEntity, entityStart := indexRune(b, 0, '&')
	// If it has no &, don't do the expensive allocations, return as-is
	if beginEntity == -1 {
		return b
	}
	// Start with of buffer of the inital bytes
	var buf bytes.Buffer
	// Grow it to fit at least the size of b
	buf.Grow(len(b))
	buf.Write(b[:beginEntity])
	// Loop forever
	for {
		// Find the end of the entity
		entityEnd, newIdx := indexRune(b, entityStart, ';')
		// If no end of entity, break out of loop, we're done
		if entityEnd == -1 {
			buf.Write(b[entityStart:])
			break
		}
		// If the element was a number, lookup
		if b[entityStart] == '#' {
			if b[entityStart+1] == 'x' {
				// Try to parse as hex, if valid use the rune value
				num := unsafeString(b[entityStart+2:entityEnd])
				if n, err := strconv.ParseInt(num, 16, 64); err == nil {
					// TODO: err check?
					buf.WriteRune(rune(n))
				} else {
					buf.Write(b[beginEntity:newIdx])
				}
			} else {
				// Try to parse the number, if valid use the rune value
				num := unsafeString(b[entityStart+1:entityEnd])
				if n, err := strconv.Atoi(num); err == nil {
					// TODO: err check?
					buf.WriteRune(rune(n))
				} else {
					buf.Write(b[beginEntity:newIdx])
				}
			}
		} else {
			// Try to look it up instead by name
			// TODO: custom Entity handling?
			name := unsafeString(b[entityStart:entityEnd])
			var replacement string
			switch name {
			case "lt": 
				replacement = "<"
			case "gt": 
				replacement = ">"
			case "amp": 
				replacement = "&"
			case "apos": 
				replacement = "'"
			case "quot":
				replacement = `"`
			default:
				replacement = xml.HTMLEntity[name]
			}
			// If we found a replacement, write that, else write the entity as-is
			if replacement != "" {
				buf.WriteString(replacement)
			} else {
				buf.Write(b[beginEntity:newIdx])
			}
		}
		// Now, find the next entity 
		beginEntity, entityStart = indexRune(b, newIdx, '&')
		// If no match, we're at the end, add the rest and break
		if beginEntity == -1 {
			buf.Write(b[newIdx:])
			break
		}
	}
	return buf.Bytes()
}

// hasPrefix does a safe check if b at offset has the provided ASCII prefix
// TODO: benchmark against bytes functions that do similar
func hasPrefix(b []byte, offset int, prefix string) bool {
	// Make sure we have enough space to check
	if len(b) - offset < len(prefix) {
		return false
	}
	// Loop each rune in prefix and check against b
	for idx, char := range prefix {
		if b[idx + offset] != byte(char) {
			return false
		}
	}
	return true
}

// splitName takes a byte slice and returns the xml.Name for it (optionally trimming whitespace)
func splitName(b []byte, trim bool) xml.Name {
	if idx := bytes.IndexRune(b, ':'); idx != -1 {
		if trim {
			// TODO: This searches left and right unnecessarily
			return xml.Name{
				Space: strings.TrimSpace(unsafeString(b[0:idx])),
				Local: strings.TrimSpace(unsafeString(b[idx+1:])),
			}
		} else {
			return xml.Name{
				Space: unsafeString(b[0:idx]),
				Local: unsafeString(b[idx+1:]),
			}
		}
	}
	if trim {
		return xml.Name{
			Local: unsafeString(b),
		}
	} else {
		return xml.Name{
			Local: strings.TrimSpace(unsafeString(b)),
		}
	}
}

// indexRune calls bytes.IndexRune starting at offset
// It returns the matched index (or -1) plus the original offset
func indexRune(b []byte, offset int, r rune) (start int, end int) {
	start = bytes.IndexRune(b[offset:], r)
	if start != -1 {
		start += offset
		end = start + 1
	} else {
		end = -1
	}
	return
}

// indexString calls bytes.Index starting at offset
// It returns the matched index (or -1) plus the original offset
func indexString(b []byte, offset int, needle string) (start int, end int) {
	start = bytes.Index(b[offset:], []byte(needle))
	if start != -1 {
		start += offset
		end = start + len(needle)
	} else {
		end = -1
	}
	return
}

// TokenReader implements xml.TokenReader
type TokenReader struct {
	// b is the raw byte slice we are parsing
	b []byte
	// idx is the offset in b we start the next token at
	idx int 
	// nextToken is used when there is a self-terminated element
	nextToken *xml.EndElement
}

// Token returns the next in b
func (tr *TokenReader) Token() (xml.Token, error) {
	// If we already have a pending token, return that and clean it up
	if tr.nextToken != nil {
		token := *tr.nextToken
		tr.nextToken = nil
		return token, nil
	}
	// If we've reached the end, bail
	if tr.idx + 1 >= len(tr.b) {
		return nil, nil
	}
	// Start at the current offset
	b := tr.b[tr.idx:]
	// Find the start of the XML element
	charEnd, elemStart := indexRune(b, 0, '<')
	// If no element found, must be EOF, set charEnd to remaining length 
	if charEnd == -1 {
		charEnd = len(b) - 1
	}
	// If the first character isn't a < we've got CharData
	if b[0] != '<' {
		// Build the CharData and return it
		// TODO: decode
		tr.idx += charEnd
		return xml.CharData(decodedValue(b[0:charEnd])), nil
	}
	// The smallest possible element is <a>, anything smaller quit 
	if rem := len(b) - elemStart; rem < 3 {
		return nil, fmt.Errorf(
			"Not enough bytes (%d) remaining for valid XML declaration", 
			rem,
		)
	}
	// Check if the next byte is a comment or declaration
	// This is safe due to above length check
	switch b[elemStart] {
	// ProcInst
	case '?':
		// Find the target of the ProcInst
		// TODO: Can other whitespace be valid here?
		targetEnd, instStart := indexRune(b, elemStart + 1, ' ')
		if targetEnd == -1 {
			return nil, fmt.Errorf(
				"Couldn't find target of XML ProcInst in: %s", 
				unsafeString(b[elemStart+1:]),
			)
		}
		// Find the end of the comment
		// TODO: Should we just search for > then backtrack for '?'?
		elemEnd, newIdx := indexString(b, elemStart+2, "?>")
		if elemEnd == -1 {
			return nil, fmt.Errorf(
				"Couldn't find end of XML ProcInst in: %s", 
				unsafeString(b[elemStart+1:]),
			)
		}
		// Build the ProcInst and return it
		tr.idx += newIdx
		return xml.ProcInst{
			Target: unsafeString(b[elemStart + 1:targetEnd]),
			Inst: b[instStart:elemEnd],
		}, nil
	// Directive
	case '!':
		// Find the end of the Directive
		elemEnd, newIdx := indexRune(b, elemStart + 1, '>')
		if elemEnd == -1 {
			return nil, fmt.Errorf(
				"Couldn't find end of XML Directive in: %s", 
				unsafeString(b[elemStart+1:]),
			)
		}
		// Comments are a special case of directive
		// This is safe due to earlier length check (?)
		if hasPrefix(b, elemStart+1, "--") {
			// Comments can have > so find "real" end (-->)
			elemEnd, newIdx = indexString(b, elemStart+4, "-->")
			// Make sure it ended with --> otherwise bail
			if elemEnd == -1 {
				return nil, fmt.Errorf(
					"Couldn't find end of XML Comment: %s", 
					unsafeString(b[elemStart+1:]),
				)
			}
			// Build the comment and return it
			tr.idx += newIdx
			return xml.Comment(b[elemStart+3:elemEnd]), nil
		}
		// CDATA is a special case of directive (becomes CharData)
		if hasPrefix(b, elemStart+1, "[CDATA[") {
			// CDATA can have > so find "real" end (]]>)
			elemEnd, newIdx = indexString(b, elemStart+8, "]]>")
			// Make sure it ended with CDATA otherwise bail
			if elemEnd == -1 {
				return nil, fmt.Errorf(
					"Couldn't find end of XML CDATA: %s", 
					unsafeString(b[elemStart+1:]),
				)
			}
			// Build the comment and return it
			tr.idx += newIdx
			return xml.CharData(b[elemStart+8:elemEnd]), nil
		}
		// Build the Directive and return it
		tr.idx += newIdx
		return xml.Directive(b[elemStart+1:elemEnd]), nil
	}
	// Check if it's the end of an element and stop here
	if b[elemStart] == '/' {
		elemEnd, newIdx := indexRune(b, elemStart + 1, '>')
		if elemEnd == -1 {
			return nil, fmt.Errorf(
				"Couldn't find end of terminating XML Element in: %s", 
				unsafeString(b[elemStart+1:]),
			)
		}
		// Build the EndElement and return it
		tr.idx += newIdx
		return xml.EndElement{
			Name: splitName(b[elemStart + 1:elemEnd], true),
		}, nil
	}
	// Must be an element if we've reached this point
	var attrs []xml.Attr
	// Find the real end of the element
	elemEnd, newIdx := indexRune(b, elemStart, '>')
	if elemEnd == -1 {
		return nil, fmt.Errorf(
			"Couldn't find end of XML Element in: %s", 
			unsafeString(b[elemStart+1:]),
		)
	}
	// Check if it has attributes (indicated by a space following the name)
	// TODO: Use a helper here
	elemNameEnd := bytes.IndexRune(b[elemStart:elemStart+elemEnd], ' ')
	if elemNameEnd != -1 {
		elemNameEnd += elemStart
	}
	// Only try to parse elements if we have values
	if elemNameEnd != -1 {
		// The start of the attribute
		startAttr := elemNameEnd + 1
		for {
			// Find the = (key=val)
			equalStart, equalEnd := indexRune(b[:elemEnd], startAttr, '=')
			// If none found, must be end of attrs, break loop
			// TODO: there could be illegal values here we should err
			if equalStart == -1 {
				break
			}
			// Find the start of the quoted value
			_, quoteStart := indexRune(b[:elemEnd], equalEnd, '"')
			if quoteStart == -1 {
				return nil, fmt.Errorf(
					"Couldn't find start of XML attribute in: %s", 
					unsafeString(b[equalEnd:]),
				)
			}
			// Find the end of the quoted value
			quoteEnd, newIdx := indexRune(b[:elemEnd], quoteStart, '"')
			if quoteEnd == -1 {
				return nil, fmt.Errorf(
					"Couldn't find end of XML attribute in: %s", 
					unsafeString(b[quoteStart:]),
				)
			}
			// Add the parsed attribute
			attrs = append(attrs, xml.Attr{
				Name: splitName(b[startAttr:equalStart], true),
				Value: unsafeString(decodedValue(b[quoteStart:quoteEnd])),
			})
			// Set the offset for the next attr loop
			startAttr = newIdx
		}
	}
	// If the element ends in /, it's self terminating
	selfTerminated := false
	if b[elemEnd-1] == '/' {
		selfTerminated = true
		if elemNameEnd == -1 {
			elemNameEnd = elemEnd-1
		}
	} else if elemNameEnd == -1 {
		elemNameEnd = elemEnd
	}
	// Calculate the name (won't include / due to above)
	name := splitName(b[elemStart:elemNameEnd], false)
	if selfTerminated {
		tr.nextToken = &xml.EndElement{
			Name: name,
		}
	}
	// Build the StartElement and return it
	tr.idx += newIdx
	return xml.StartElement{
		Name: name,
		Attr: attrs,
	}, nil
}

// NewTokenReader creates a *TokenReader instance given b (which should not be modified once passed into TokenReader)
func NewTokenReader(b []byte) *TokenReader {
	return &TokenReader{
		b: b,
		// Start at 0 index
		idx: 0,
	}
}
