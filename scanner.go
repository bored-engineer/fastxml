package fastxml

import (
	"bytes"
	"errors"
	"io"
)

// Allocate the errors once and return the same structs
var (
	errCDATASuffix   = errors.New("expected Token to end with ']]>'")
	errElementSuffix = errors.New("expected Token to end with '>'")
)

// Allocate these once instead of on each bytes.Index/HasPrefix/HasSuffix call
var (
	prefixCDATA = []byte("<![CDATA[")
	suffixCDATA = []byte("]]>")
)

// Scanner reads a []byte emitting each "token" as a slice
type Scanner struct {
	buf []byte // immutable slice of data
	pos int    // pos is the current offset in buf
}

// Offset outputs the internal position the Scanner is at
func (s *Scanner) Offset() int {
	return s.pos
}

// Seek implements the io.Seeker interface
func (s *Scanner) Seek(offset int64, whence int) (int64, error) {
	var abs int
	switch whence {
	case io.SeekStart:
		abs = int(offset)
	case io.SeekCurrent:
		abs = s.pos + int(offset)
	case io.SeekEnd:
		abs = len(s.buf) + int(offset)
	default:
		return int64(s.pos), errors.New("invalid whence")
	}
	if abs < 0 {
		return int64(s.pos), errors.New("negative position")
	} else if abs > len(s.buf) {
		return int64(s.pos), errors.New("seek past end of buffer")
	}
	s.pos = abs
	return int64(s.pos), nil
}

// Next produces the next token from the scanner
// When no more tokens are available io.EOF is returned AND the trailing token (if any)
func (s *Scanner) Next() (token []byte, chardata bool, err error) {
	// EOF, no more data
	if s.pos == len(s.buf) {
		err = io.EOF
		return
	}
	// Find the next (potential) element start
	// Doing a lookup on first byte avoids a duplicate call to bytes.IndexByte
	if s.buf[s.pos] != '<' {
		next := bytes.IndexByte(s.buf[s.pos+1:], '<')
		// If we are at the EOF
		if next == -1 {
			// Trailing CharData returned here if present
			if s.pos < len(s.buf) {
				token = s.buf[s.pos:]
				s.pos = len(s.buf)
				chardata = true
				return
			}
			err = io.EOF
			return
		}
		// If there's a gap between next and current pos, that's CharData
		next++ // account for the +1 in IndexByte
		token = s.buf[s.pos : s.pos+next]
		s.pos += next
		chardata = true
		return
	}
	// If it starts with the CDATA prefix it's actually CharData (special case)
	if bytes.HasPrefix(s.buf[s.pos:], prefixCDATA) {
		chardata = true
		// Find the end of the CDATA section
		end := bytes.Index(s.buf[s.pos+8:], suffixCDATA)
		if end == -1 {
			token = s.buf[s.pos:]
			err = errCDATASuffix
			return
		}
		end += 11 // len(prefixCDATA) + len(suffixCDATA)
		token = s.buf[s.pos : s.pos+end]
		s.pos += end
		return
	}
	// Find the end of the element
	end := bytes.IndexByte(s.buf[s.pos:], '>')
	if end == -1 {
		token = s.buf[s.pos:]
		err = errElementSuffix
		return
	}
	end++ // len('>')
	token = s.buf[s.pos : s.pos+end]
	s.pos += end
	return
}

// Skip will skip until the end of the most recently processed element
func (s *Scanner) Skip() error {
	for depth := 1; depth > 0; {
		// Grab the next token, bail on error
		token, chardata, err := s.Next()
		if err != nil {
			return err
		}
		// Skip ProcInst, Directive, CharData
		if chardata || !IsElement(token) {
			continue
		}
		// If self-closing, has no impact on depth
		if IsSelfClosing(token) {
			continue
		}
		// Increment the depth based on an element start/stop
		if IsEndElement(token) {
			depth--
		} else {
			depth++
		}
	}
	return nil
}

// Reset replaces the buf in scanner to a new slice
func (s *Scanner) Reset(buf []byte) {
	s.buf = buf
	s.pos = 0
}

// NewScanner creates a *Scanner for a given byte slice
func NewScanner(buf []byte) *Scanner {
	return &Scanner{buf: buf, pos: 0}
}
