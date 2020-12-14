package fastxml

// IsComment determines if a Directive is a comment (<!--)
func IsComment(token []byte) bool {
	return len(token) > 4 && token[0] == '<' && token[1] == '!' && token[2] == '-' && token[3] == '-'
}

// Comment extracts the contents of a comment
func Comment(token []byte) []byte {
	if len(token) <= 7 {
		return nil
	}
	return token[4 : len(token)-3]
}
