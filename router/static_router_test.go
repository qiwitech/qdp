package router

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticRouter(t *testing.T) {
	r := NewStatic("first")

	r.SetNodes([]string{"0=first", "10=second", "78=last"})
	assert.Equal(t, []string{"0=first", "10=second", "78=last"}, r.Nodes())

	assert.Equal(t, "first", r.GetHostByKey("0"))
	assert.Equal(t, "first", r.GetHostByKey("5"))
	assert.Equal(t, "first", r.GetHostByKey("9"))
	assert.Equal(t, "second", r.GetHostByKey("10"))
	assert.Equal(t, "second", r.GetHostByKey("11"))
	assert.Equal(t, "second", r.GetHostByKey("77"))
	assert.Equal(t, "last", r.GetHostByKey("78"))
	assert.Equal(t, "last", r.GetHostByKey("200"))

	assert.Panics(t, func() { assert.Equal(t, "last", r.GetHostByKey("incorrect_id")) })

	// missed ids is calculated as i * 1/MaxUint64
	//	assert.Panics(t, func() { r.SetNodes([]string{"0=first", "10=second", "78=last", "invalid"}) })
	assert.Panics(t, func() { r.SetNodes([]string{"0=first", "10=second", "78=last", "aa=invalid-shard"}) })

	assert.True(t, r.IsSelf("first"))
	assert.False(t, r.IsSelf("last"))
}

func TestStaticRouterCase1(t *testing.T) {
	r := NewStatic("10.0.9.2:31337")

	r.SetNodes(strings.Split("0=10.0.9.2:31337,1024819115206086201=10.0.9.3:31337,2049638230412172402=10.0.9.4:31337,3074457345618258603=10.0.9.5:31337,4099276460824344804=10.0.9.6:31337,5124095576030431005=10.0.9.7:31337,6148914691236517206=10.0.9.8:31337,7173733806442603407=10.0.9.9:31337,8198552921648689608=10.0.9.10:31337", ","))

	assert.Equal(t, "10.0.9.2:31337", r.GetHostByKey("0"))
	assert.Equal(t, "10.0.9.2:31337", r.GetHostByKey("1"))
	assert.Equal(t, "10.0.9.2:31337", r.GetHostByKey("1000000"))
	assert.Equal(t, "10.0.9.3:31337", r.GetHostByKey("1024819115206086201"))
	assert.Equal(t, "10.0.9.3:31337", r.GetHostByKey("1024819115206086202"))

	assert.Equal(t, "10.0.9.10:31337", r.GetHostByKey(fmt.Sprintf("%d", uint64(math.MaxUint64))))
}

func TestStaticRouterEqualRanges(t *testing.T) {
	r := NewStatic("")

	r.SetNodes([]string{"a", "b", "c", "d"})

	assert.Equal(t, "a", r.GetHostByKey("0"))
	assert.Equal(t, "a", r.GetHostByKey("1"))
	assert.Equal(t, "a", r.GetHostByKey("100000000"))
	assert.Equal(t, "a", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4-1))))

	assert.Equal(t, "b", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4))))
	assert.Equal(t, "b", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4+1))))
	assert.Equal(t, "b", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4+100000000000))))
	assert.Equal(t, "b", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/2-1))))

	assert.Equal(t, "c", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/2))))
	assert.Equal(t, "c", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/2+1))))
	assert.Equal(t, "c", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/2+100000000))))
	assert.Equal(t, "c", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4*3-1))))

	assert.Equal(t, "d", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4*3))))
	assert.Equal(t, "d", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4*3+1))))
	assert.Equal(t, "d", r.GetHostByKey(fmt.Sprintf("%d", ^uint64(0)-1)))
	assert.Equal(t, "d", r.GetHostByKey(fmt.Sprintf("%d", ^uint64(0))))
}

func TestStaticRouterNotEqualRanges(t *testing.T) {
	r := NewStatic("")

	r.SetNodes([]string{"a", "b", "", "d"}) // b have half of the range and a and d have quarters

	assert.Equal(t, "a", r.GetHostByKey("0"))
	assert.Equal(t, "a", r.GetHostByKey("1"))
	assert.Equal(t, "a", r.GetHostByKey("100000000"))
	assert.Equal(t, "a", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4-1))))

	assert.Equal(t, "b", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4))))
	assert.Equal(t, "b", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4+1))))
	assert.Equal(t, "b", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4+100000000000))))
	assert.Equal(t, "b", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/2-1))))

	assert.Equal(t, "b", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/2))))
	assert.Equal(t, "b", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/2+1))))
	assert.Equal(t, "b", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/2+100000000))))
	assert.Equal(t, "b", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4*3-1))))

	assert.Equal(t, "d", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4*3))))
	assert.Equal(t, "d", r.GetHostByKey(fmt.Sprintf("%d", uint64((1<<64)/4*3+1))))
	assert.Equal(t, "d", r.GetHostByKey(fmt.Sprintf("%d", ^uint64(0)-1)))
	assert.Equal(t, "d", r.GetHostByKey(fmt.Sprintf("%d", ^uint64(0))))
}
