package fastxml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsProcInst(t *testing.T) {
	assert.True(t, IsProcInst([]byte("<?target inst?>")))
	assert.False(t, IsProcInst([]byte("<element>")))
}
func TestProcInst(t *testing.T) {
	target, inst := ProcInst([]byte("<?target inst?>"))
	assert.Equal(t, "target", string(target))
	assert.Equal(t, "inst", string(inst))
	target, inst = ProcInst([]byte("<?invalid?>"))
	assert.Equal(t, "invalid", string(target))
	assert.Nil(t, inst)
}
