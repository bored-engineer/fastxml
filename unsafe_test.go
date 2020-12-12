package fastxml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_unsafeString(t *testing.T) {
	source := []byte("lorem ipsum dolor sit amet")
	assert.Equal(t, "ipsum dolor", unsafeString(source[6:17]))
}
