package fastxml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDirective(t *testing.T) {
	assert.True(t, IsDirective([]byte("<!text>")))
	assert.False(t, IsDirective([]byte("<element>")))
}

func TestDirective(t *testing.T) {
	dir := Directive([]byte("<!text>"))
	assert.Equal(t, "text", string(dir))
}
