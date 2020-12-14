package fastxml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsComment(t *testing.T) {
	assert.True(t, IsComment([]byte("<!--comment-->")))
	assert.False(t, IsComment([]byte("<!directive>")))
}

func TestComment(t *testing.T) {
	comment := Comment([]byte("<!--hello world-->"))
	assert.Equal(t, "hello world", string(comment))
}
