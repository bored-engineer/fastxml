package fastxml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_String(t *testing.T) {
	source := []byte("lorem ipsum dolor sit amet")
	assert.Equal(t, "ipsum dolor", String(source[6:17]))
}
