package fastxml

import "unsafe"

// unsafeString performs an _unsafe_ no-copy string allocation from bs
// https://github.com/golang/go/issues/25484 has more info on this
// the implementation is roughly taken from strings.Builder's
func unsafeString(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}
