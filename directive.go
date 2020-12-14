package fastxml

// IsDirective determines if a []byte is directive (ex: <!text>)
func IsDirective(b []byte) bool {
	return len(b) >= 4 && b[0] == '<' && b[1] == '!' && b[2] != '-' && b[3] != '-'
}

// Directive returns the contents of a directive (ex: `<!text>` -> `text`)
func Directive(b []byte) []byte {
	if len(b) < 3 {
		return nil
	}
	return b[2 : len(b)-1]
}
