package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelfRoute(t *testing.T) {
	r := New("me")
	ns := r.Nodes()
	assert.Equal(t, []string{}, ns)

	res := r.GetHostByKey("some_key")
	assert.Equal(t, "", res)
}

func TestIsSelf(t *testing.T) {
	r := New("me")
	assert.True(t, r.IsSelf("me"))
	assert.False(t, r.IsSelf("not-me"))
}

func TestSetNodes(t *testing.T) {
	r := New("me")
	r.SetNodes([]string{"host1", "host2"})
	assert.Equal(t, []string{"host1", "host2"}, r.Nodes())
}
