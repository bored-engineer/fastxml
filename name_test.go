package fastxml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	space, local := Name([]byte("foo"))
	assert.Nil(t, space)
	assert.Equal(t, []byte("foo"), local)
	space, local = Name([]byte("space:local"))
	assert.Equal(t, []byte("space"), space)
	assert.Equal(t, []byte("local"), local)
}
