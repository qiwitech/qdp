package processor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/chain"
	"github.com/qiwitech/qdp/pt"
)

func TestMultiProcessSettings(t *testing.T) {
	p := NewSettingsMultiprocessor(chain.NewSettingsChain(), 3)

	s := &pt.Settings{Account: 10}
	res, err := p.ProcessSettings(context.TODO(), s)
	assert.NoError(t, err)

	assert.Equal(t, pt.SettingsResult{
		SettingsID: pt.NewSettingsID(10, 1),
		Hash:       pt.HashFromString("74280ac6f6059268431efe2a9b8903957a08f7685e651c6cbbedcd83c101bcc5"),
	}, res)
}

func TestMultiGetLastSettings(t *testing.T) {
	c := chain.NewSettingsChain()
	p := NewSettingsMultiprocessor(c, 3)

	b, err := p.GetLastSettings(context.TODO(), pt.AccID(10))
	assert.NoError(t, err)
	assert.Equal(t, (*pt.Settings)(nil), b)
}
