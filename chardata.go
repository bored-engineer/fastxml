package fastxml

import "bytes"

// CharData will output the decoded CharData
func CharData(token []byte) ([]byte, error) {
	// CDATA is returned as-is without decoding
	if bytes.HasPrefix(token, prefixCDATA) && bytes.HasSuffix(token, suffixCDATA) {
		// token[len(prefixCDATA):len(token) - len(suffixCDATA)]
		return token[9 : len(token)-3], nil
	}
	// Decode the entities
	return DecodeEntities(token)
}
