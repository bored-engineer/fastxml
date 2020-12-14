package fastxml

import "bytes"

// Name produces the space and local values given a name (ex: `foo:bar` -> (`foo`, `bar`))
func Name(token []byte) (space []byte, local []byte) {
	if idx := bytes.IndexByte(token, ':'); idx != -1 {
		return token[:idx], token[idx+1:]
	}
	return nil, token
}
