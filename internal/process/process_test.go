package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProcess(t *testing.T) {
	p := NewProcess()
	assert.NotNil(t, p)

	assert.Equal(t, 0, len(p.procList))

	err := p.ListProcesses()
	assert.NoError(t, err)
	assert.NotEmpty(t, p.procList)
}
