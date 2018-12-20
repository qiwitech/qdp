package chain

import (
	"math/rand"
	"sync"

	"github.com/qiwitech/qdp/pt"
)

const skiplistMaxLevel = 32
const skiplistBuiltInHeight = 4

var skiplistElementsPool = sync.Pool{
	New: func() interface{} {
		return &element{}
	},
}

func newSLElement(k pt.ID, h int) *element {
	v := skiplistElementsPool.Get()
	e := v.(*element)
	*e = element{
		key:    k,
		height: byte(h),
	}
	if h > len(e.next) {
		e.more = make([]*element, h-len(e.next))
	}
	return e
}
func putSLElement(e *element) {
	for i := range e.next {
		e.next[i] = nil
	}
	e.more = nil
	skiplistElementsPool.Put(e)
}

func newSkipList() *skiplist {
	l := &skiplist{}
	l.zero = element{
		key:    0,
		height: skiplistMaxLevel,
		more:   l.zeromore[:],
	}
	return l
}

type skiplist struct {
	zero     element
	zeromore [skiplistMaxLevel - skiplistBuiltInHeight]*element
}

type element struct {
	key    pt.ID
	next   [skiplistBuiltInHeight]*element
	more   []*element
	Value  chainElement
	height byte
}

func (l *skiplist) Front() *element {
	return l.zero.next[0]
}

// TODO(outself): rename
func (l *skiplist) GetOrInsert(k pt.ID) *element {
	return l.findOrInsert(k)
}

func (l *skiplist) Get(k pt.ID) *element {
	cur, _ := l.findEl(k)
	if cur != nil && cur.key != k {
		return nil
	}
	return cur
}

// Deprecated: use direct DB getters instead
func (l *skiplist) FindLessEqual(k pt.ID) *element {
	cur, par := l.findEl(k)
	if cur != nil {
		return cur
	}
	return par
}

func (l *skiplist) findEl(k pt.ID) (*element, *element) {
	var par *element
	cur := l.zero.next[0]
	for cur != nil && cur.key > k {
		par = cur
		cur = cur.furtherLessEqual(k)
	}
	return cur, par
}

func (l *skiplist) findOrInsert(key pt.ID) *element {
	var update [skiplistMaxLevel]*element

	var par *element
	cur := &l.zero
	for {
		par = cur
		cur = cur.furtherLessEqual(key)
		for i := cur.Height(); i < par.Height(); i++ {
			update[i] = par
		}
		//	log.Printf("cur %v", cur.ToString())
		if cur == nil || cur.key <= key {
			break
		}
	}

	if cur != nil {
		return cur
	}

	h := l.randHeight()
	cur = newSLElement(key, h)
	for i := 0; i < h; i++ {
		link := update[i].NextLevel(i)
		cur.setNextLevel(i, link)
		update[i].setNextLevel(i, cur)
	}

	return cur
}

func (l *skiplist) Remove(key pt.ID) {
	var update [skiplistMaxLevel]*element

	var par *element
	cur := &l.zero
	for {
		par = cur
		cur = cur.furtherLess(key)
		for i := cur.Height(); i < par.Height() && i < len(update); i++ {
			update[i] = par
		}
		if cur == nil {
			break
		}
	}

	cur = par.Next()
	if cur == nil || cur.key < key {
		return
	}

	i := 0
	for ; i < cur.Height() && i < par.Height(); i++ {
		par.setNextLevel(i, cur.NextLevel(i))
	}
	for ; i < cur.Height(); i++ {
		update[i].setNextLevel(i, cur.NextLevel(i))
	}
	putSLElement(cur)
}

func (l *skiplist) RemoveAfter(key pt.ID) {
	var update [skiplistMaxLevel]*element

	var par *element
	cur := &l.zero
	for {
		par = cur
		cur = cur.furtherLessEqual(key)
		for i := cur.Height(); i < par.Height() && i < len(update); i++ {
			update[i] = par
		}
		if cur == nil || cur.key < key {
			break
		}
	}

	cur = par.next[0]
	for i := range update {
		update[i].setNextLevel(i, nil)
	}
	for cur != nil {
		par = cur
		cur = cur.next[0]
		//	putSLElement(par)
	}
}

func (l *skiplist) randHeight() int {
	rnd := rand.Int63()
	h := 1
	for rnd&1 == 1 && h < int(l.zero.height) {
		h++
		rnd >>= 1
	}

	return h
}

func (e *element) Next() *element {
	return e.next[0]
}

func (e *element) NextLevel(l int) *element {
	if l >= len(e.next) {
		l -= len(e.next)
		return e.more[l]
	}
	return e.next[l]
}

func (e *element) setNextLevel(l int, v *element) {
	if l >= len(e.next) {
		l -= len(e.next)
		e.more[l] = v
		return
	}
	e.next[l] = v
}

func (e *element) furtherLessEqual(k pt.ID) *element {
	if len(e.more) > 0 {
		for i := len(e.more) - 1; i >= 0; i-- {
			next := e.more[i]
			if next == nil {
				continue
			}
			if next.key >= k {
				return next
			}
		}
	}
	for i := skiplistBuiltInHeight - 1; i >= 0; i-- {
		next := e.next[i]
		if next == nil {
			continue
		}
		if next.key >= k {
			return next
		}
	}

	return nil
}

func (e *element) furtherLess(k pt.ID) *element {
	if len(e.more) > 0 {
		for i := len(e.more) - 1; i >= 0; i-- {
			next := e.more[i]
			if next == nil {
				continue
			}
			if next.key > k {
				return next
			}
		}
	}
	for i := skiplistBuiltInHeight - 1; i >= 0; i-- {
		next := e.next[i]
		if next == nil {
			continue
		}
		if next.key > k {
			return next
		}
	}

	return nil
}

func (e *element) Height() int {
	if e == nil {
		return 0
	}
	return int(e.height)
}
