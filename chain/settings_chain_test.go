package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/pt"
)

func TestPutInvalidSettings(t *testing.T) {
	c := NewSettingsChain()

	assert.Panics(t, func() {
		c.Put(&pt.Settings{})
	})
}

func TestPutSettings(t *testing.T) {
	c := NewSettingsChain()

	c.Put(&pt.Settings{ID: 1, Account: 10})
	assert.Equal(t, &pt.Settings{ID: 1, Account: 10}, c.GetLastSettings(10))
}

func TestEmptyGetLastSettings(t *testing.T) {
	c := NewSettingsChain()
	assert.Nil(t, c.GetLastSettings(10))
}

func TestGetLastSettingsHash(t *testing.T) {
	c := NewSettingsChain()
	assert.Equal(t, pt.ZeroHash, c.GetLastHash(10))
	c.Put(&pt.Settings{ID: 1, Account: 10, Hash: pt.HashFromString("1234")})
	assert.Equal(t, pt.HashFromString("1234"), c.GetLastHash(10))
}
