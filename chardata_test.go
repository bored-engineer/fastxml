package fastxml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCharData(t *testing.T) {
	data, err := CharData([]byte("hello &amp; world"))
	assert.NoError(t, err)
	assert.Equal(t, "hello & world", string(data))
	data, err = CharData([]byte("<![CDATA[<complex &amp;]]>"))
	assert.NoError(t, err)
	assert.Equal(t, "<complex &amp;", string(data))
	_, err = CharData([]byte("&invalid;"))
	assert.Error(t, err)
}
