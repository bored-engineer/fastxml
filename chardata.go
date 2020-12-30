package fastxml

import "bytes"

// CharData will output the decoded CharData
func CharData(charToken []byte, scratch []byte) ([]byte, error) {
	// CDATA is returned as-is without decoding
	if bytes.HasPrefix(charToken, prefixCDATA) && bytes.HasSuffix(charToken, suffixCDATA) {
		// token[len(prefixCDATA):len(token) - len(suffixCDATA)]
		return charToken[9 : len(charToken)-3], nil
	}
	// Decode the entities
	return DecodeEntities(charToken, scratch)
}

// CharDataAppend will efficiently append the decoded CharData to the output slice
func CharDataAppend(out []byte, charToken []byte) ([]byte, error) {
	// CDATA is appended as-is without decoding
	if bytes.HasPrefix(charToken, prefixCDATA) && bytes.HasSuffix(charToken, suffixCDATA) {
		// token[len(prefixCDATA):len(token) - len(suffixCDATA)]
		// memmove which will be faster
		return append(out, charToken[9:len(charToken)-3]...), nil
	}
	return DecodeEntitiesAppend(out, charToken)
}
