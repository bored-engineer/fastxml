package fastxml

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"
)

// decodeEntities appends to scratch
func decodeEntities(scratch []byte, in []byte, start int) ([]byte, error) {
	scratch = append(scratch, in[:start]...)
	start++
	for {
		// Find the end of the entity
		end := bytes.IndexRune(in[start:], ';')
		if end == -1 {
			return in, errors.New("expected ';' to end XML entity, not found")
		}
		// rune based on hex/decimal value
		if in[start] == '#' {
			offset := start + 1
			base := 10
			if in[start+1] == 'x' {
				base = 16
				offset++
			}
			str := String(in[offset : start+end])
			// rune is a int32
			num, err := strconv.ParseInt(str, base, 32)
			if err != nil {
				return in, fmt.Errorf("failed to decode %q: %w", str, err)
			}
			// Make room for utf8.UTFMax if needed before hitting capacity
			size := len(scratch)
			if cap(scratch) >= size+utf8.UTFMax {
				scratch = append(scratch, make([]byte, utf8.UTFMax)...)
			}
			// Encode in place
			size += utf8.EncodeRune(scratch[size:size+utf8.UTFMax], rune(num))
			scratch = scratch[:size]
		} else {
			// Lookup an entity by name
			entity := String(in[start : start+end])
			// common entities are in the switch before hashmap
			switch entity {
			case "lt":
				scratch = append(scratch, '<')
			case "gt":
				scratch = append(scratch, '>')
			case "amp":
				scratch = append(scratch, '&')
			case "apos":
				scratch = append(scratch, '\'')
			case "quot":
				scratch = append(scratch, '"')
			default:
				// Check from more expensive map
				decoded, ok := xml.HTMLEntity[entity]
				if !ok {
					return in, fmt.Errorf("unknown XML entity %q", entity)
				}
				scratch = append(scratch, decoded...)
			}
		}
		// Find next entity
		if idx := bytes.IndexRune(in[start+end:], '&'); idx != -1 {
			start += end + idx + 1
		} else {
			// No more entities, copy rest of bytes and return
			scratch = append(scratch, in[start+end+1:]...)
			return scratch, nil
		}
	}
}

// DecodeEntities will resolve any (known) XML entities in the input
// scratch is an optional existing byte slice to append the decoded
// values to. If scratch is nil a new slice will be allocated
func DecodeEntities(in []byte, scratch []byte) ([]byte, error) {
	start := bytes.IndexRune(in, '&')
	if start == -1 {
		// No entities, return as-is
		return in, nil
	}
	// If no scratch slice given allocate a new one with the "right" capacity
	if scratch == nil {
		// The final result will always be smaller than the input length
		scratch = make([]byte, 0, len(in))
	}
	return decodeEntities(scratch, in, start)
}

// DecodeEntitiesAppend will efficiently append the decoded in to out
// Behaves the same as DecodeEntities
func DecodeEntitiesAppend(out []byte, in []byte) ([]byte, error) {
	start := bytes.IndexRune(in, '&')
	if start == -1 {
		// No entities, memmove as-is (fast)
		return append(out, in...), nil
	}
	return decodeEntities(out, in, start)
}
