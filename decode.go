package fastxml

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"
)

// DecodeEntities will resolve any (known) XML entities in the input
func DecodeEntities(in []byte) ([]byte, error) {
	start := bytes.IndexRune(in, '&')
	if start == -1 {
		// No entities, return as-is
		return in, nil
	}
	// The final result will always be smaller than the input length
	buf := make([]byte, len(in))
	size := copy(buf, in[:start])
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
			// TODO: This is probably a good bit slow
			size += utf8.EncodeRune(buf[size:], rune(num))
		} else {
			// Lookup an entity by name
			entity := String(in[start : start+end])
			// common entities are in the switch before hashmap
			switch entity {
			case "lt":
				buf[size] = '<'
				size++
			case "gt":
				buf[size] = '>'
				size++
			case "amp":
				buf[size] = '&'
				size++
			case "apos":
				buf[size] = '\''
				size++
			case "quot":
				buf[size] = '"'
				size++
			default:
				// Check from more expensive map
				decoded, ok := xml.HTMLEntity[entity]
				if !ok {
					return in, fmt.Errorf("unknown XML entity %q", entity)
				}
				size += copy(buf[size:], decoded)
			}
		}
		// Find next entity
		if idx := bytes.IndexRune(in[start+end:], '&'); idx != -1 {
			start += end + idx + 1
		} else {
			// No more entities, copy rest of bytes and return
			size += copy(buf[size:], in[start+end+1:])
			return buf[:size], nil
		}
	}
}
