package chain

import (
	"math/rand"
	"sync"

	"github.com/qiwitech/qdp/pt"
)

const settingsSkiplistMaxLevel = 32
const settingsSkiplistBuiltInHeight = 4

var skiplistSettingsElementsPool = sync.Pool{
	New: func() interface{} {
		return &settingsElement{}
	},
}

func newSLSettingsElement(k pt.ID, h int) *settingsElement {
	v := skiplistSettingsElementsPool.Get()
	e := v.(*settingsElement)
	*e = settingsElement{
		key:    k,
		height: byte(h),
	}
	if h > len(e.next) {
		e.more = make([]*settingsElement, h-len(e.next))
	}
	return e
}
func putSLSettingsElement(e *settingsElement) {
	for i := range e.next {
		e.next[i] = nil
	}
	e.more = nil
	skiplistSettingsElementsPool.Put(e)
}

func newSettingsSkipList() *settingsSkiplist {
	l := &settingsSkiplist{}
	l.zero = settingsElement{
		key:    0,
		height: settingsSkiplistMaxLevel,
		more:   l.zeromore[:],
	}
	return l
}

type settingsSkiplist struct {
	zero     settingsElement
	zeromore [settingsSkiplistMaxLevel - settingsSkiplistBuiltInHeight]*settingsElement
}

type settingsElement struct {
	key      pt.ID
	next     [settingsSkiplistBuiltInHeight]*settingsElement
	more     []*settingsElement
	Settings *pt.Settings
	height   byte
}

func (l *settingsSkiplist) Front() *settingsElement {
	return l.zero.next[0]
}

// TODO(outself): rename
func (l *settingsSkiplist) GetOrInsert(k pt.ID) *settingsElement {
	return l.findOrInsert(k)
}

func (l *settingsSkiplist) Get(k pt.ID) *settingsElement {
	cur, _ := l.findEl(k)
	if cur != nil && cur.key != k {
		return nil
	}
	return cur
}

// Deprecated: use direct DB getters instead
func (l *settingsSkiplist) FindLessEqual(k pt.ID) *settingsElement {
	cur, par := l.findEl(k)
	if cur != nil {
		return cur
	}
	return par
}

func (l *settingsSkiplist) findEl(k pt.ID) (*settingsElement, *settingsElement) {
	var par *settingsElement
	cur := l.zero.next[0]
	for cur != nil && cur.key > k {
		par = cur
		cur = cur.furtherLessEqual(k)
	}
	return cur, par
}

func (l *settingsSkiplist) findOrInsert(key pt.ID) *settingsElement {
	var update [settingsSkiplistMaxLevel]*settingsElement

	var par *settingsElement
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
	cur = newSLSettingsElement(key, h)
	for i := 0; i < h; i++ {
		link := update[i].NextLevel(i)
		cur.setNextLevel(i, link)
		update[i].setNextLevel(i, cur)
	}

	return cur
}

func (l *settingsSkiplist) Remove(key pt.ID) {
	var update [settingsSkiplistMaxLevel]*settingsElement

	var par *settingsElement
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
	putSLSettingsElement(cur)
}

func (l *settingsSkiplist) RemoveAfter(key pt.ID) {
	var update [settingsSkiplistMaxLevel]*settingsElement

	var par *settingsElement
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
		//	putSLSettingsElement(par)
	}
}

func (l *settingsSkiplist) randHeight() int {
	rnd := rand.Int63()
	h := 1
	for rnd&1 == 1 && h < int(l.zero.height) {
		h++
		rnd >>= 1
	}

	return h
}

func (e *settingsElement) Next() *settingsElement {
	return e.next[0]
}

func (e *settingsElement) NextLevel(l int) *settingsElement {
	if l >= len(e.next) {
		l -= len(e.next)
		return e.more[l]
	}
	return e.next[l]
}

func (e *settingsElement) setNextLevel(l int, v *settingsElement) {
	if l >= len(e.next) {
		l -= len(e.next)
		e.more[l] = v
		return
	}
	e.next[l] = v
}

func (e *settingsElement) furtherLessEqual(k pt.ID) *settingsElement {
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
	for i := settingsSkiplistBuiltInHeight - 1; i >= 0; i-- {
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

func (e *settingsElement) furtherLess(k pt.ID) *settingsElement {
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
	for i := settingsSkiplistBuiltInHeight - 1; i >= 0; i-- {
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

func (e *settingsElement) Height() int {
	if e == nil {
		return 0
	}
	return int(e.height)
}
