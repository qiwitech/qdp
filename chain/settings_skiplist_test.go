package chain

import (
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/qiwitech/qdp/pt"
)

func TestSettingsSkiplist(t *testing.T) {
	l := newSettingsSkipList()

	ids := map[pt.ID]struct{}{}

	l.GetOrInsert(1).Settings = &pt.Settings{ID: 1}
	l.GetOrInsert(3).Settings = &pt.Settings{ID: 3}
	l.GetOrInsert(2).Settings = &pt.Settings{ID: 2}
	l.GetOrInsert(4).Settings = &pt.Settings{ID: 4}
	l.GetOrInsert(7).Settings = &pt.Settings{ID: 7}

	ids[1] = struct{}{}
	ids[3] = struct{}{}
	ids[2] = struct{}{}
	ids[4] = struct{}{}
	ids[7] = struct{}{}

	var e *settingsElement
	e = l.Get(3)
	if e == nil || e.Settings.ID != 3 {
		t.Errorf("Get error, get %v, expect id 3", e)
	}
	e = l.Get(5)
	if e != nil {
		t.Errorf("Get error, get %v, expect nil", e)
	}

	e = l.FindLessEqual(5)
	if e == nil || e.Settings.ID != 7 {
		t.Errorf("FindLessEqual error, get %v, expect id 7", e)
	}

	/*
		e = l.FindValue(func(iel interface{}) bool {
			return iel.(*AccElement).Txn.ID < 2
		})
		if e == nil || e.Value == nil || e.Value.(*AccElement).Txn.ID != 1 {
			t.Errorf("FindValue error, get %v, expect id 1", e)
		}
	*/

	l.Remove(4)
	e = l.Get(4)
	if e != nil {
		t.Errorf("Remove error, get %v, expect nil (we remove 4)", e)
	}

	/*	t.Logf("tree 1.1  removed 4")
		e = &l.zero
		for e != nil {
			sender := uint64(0)
			if elval, ok := e.Value.(*ChainElement); ok {
				sender = elval.Txn.Sender
			}
			t.Logf("list el %2v %3v : %v\n", e.key, sender, e.ToString())
			e = e.Next()
		}
	*/

	l.RemoveAfter(3)
	e = l.Get(3)
	if e == nil {
		t.Errorf("RemoveAfter error, get nil, expect 3 id")
	} else if e.next[0] != nil {
		t.Errorf("RemoveAfter error, we didn't remove tail")
	}

	delete(ids, 4)
	delete(ids, 2)
	delete(ids, 1)

	for id := range ids {
		e = l.FindLessEqual(id)
		if id != e.key {
			t.Errorf("find less or equal (decreasing order): search %v got %v (key may not exists)", id, e.key)
		}
	}

	for i := 0; i < 20; i++ {
		id := pt.ID(rand.Int63n(15))
		//	log.Printf("QWEQWE %v", i)
		l.GetOrInsert(id).Settings = &pt.Settings{
			ID: id,
		}
		ids[id] = struct{}{}
	}

	for i := 0; i < 100000; i++ {
		l.GetOrInsert(pt.ID(rand.Int63n(1000))).Settings = &pt.Settings{
			ID: pt.ID(i),
		}
	}
}

func TestSettingsSkiplistRandomAddRemove(t *testing.T) {
	N := 10
	ids := map[pt.ID]struct{}{}
	l := newSettingsSkipList()
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < N; i++ {
		id := pt.ID(rnd.Intn(N))
		_, ok := ids[id]
		//	t.Logf("add %v, it was %v", id, ok)
		e := l.Get(id)
		ok2 := e != nil && e.key == id
		if ok != ok2 {
			t.Errorf("we had but not had id %v: %v", id, e)
		}

		l.GetOrInsert(id)
		ids[id] = struct{}{}
	}

	for i := 0; i < N; i++ {
		id := pt.ID(rnd.Intn(N))
		l.Remove(id)
		e := l.Get(id)
		ok2 := e != nil && e.key == id
		//	t.Logf("remove %v, it exists %v", id, ok2)
		if ok2 {
			t.Errorf("we remove %d but it still here: %v", id, e)
		}
		delete(ids, id)
	}

	s := make([]pt.ID, 0, len(ids))
	for id := range ids {
		s = append(s, id)
	}

	sort.Slice(s, func(i, j int) bool {
		return s[i] > s[j]
	})

	e := l.Front()
	iserr := false
	for _, id := range s {
		if e == nil || e.key != id {
			iserr = true
			break
		}
		e = e.Next()
	}

	if iserr {
		t.Errorf("result list is wrong")
		t.Errorf("must be")
		for _, id := range s {
			t.Errorf("   %v", id)
		}
		t.Errorf("we have")
		e = l.Front()
		for e != nil {
			t.Errorf("   %v", e.key)
			e = e.Next()
		}
	}
}

func TestSettingsSkiplistRandHeight(t *testing.T) {
	l := newSettingsSkipList()
	var cnt [skiplistMaxLevel]int

	for i := 0; i < 10000; i++ {
		h := l.randHeight()
		cnt[h]++
	}

	for i := range cnt {
		t.Logf("lev: %v = %v", i, cnt[i])
		if i < 2 { // start from i=2 (pair 1,2). cnt[0] == 0 always
			continue
		}
		if cnt[i] == 0 {
			break
		}
		if cnt[i] < cnt[i-1]/4 || cnt[i] > cnt[i-1] {
			t.Errorf("too big dispersion. at step %d: %d must be about half of %d", i, cnt[i], cnt[i-1])
		}
	}
}

func TestSettingsSkiplistRemoveAfter(t *testing.T) {
	M := 100
	N := 100
	for j := 0; j < M; j++ {
		l := newSettingsSkipList()
		for i := 0; i < N; i++ {
			l.GetOrInsert(pt.ID(i))
		}
		//	t.Logf("list\n%v", l.Dump())
		r := pt.ID(rand.Intn(N))
		l.RemoveAfter(r)

		//	t.Logf("removed after %v\n%v", r, l.Dump())

		e := l.Front()
		for e.Next() != nil {
			e = e.Next()
		}
		if e.key != r {
			t.Errorf("we removed after %v but last el is %v", r, e)
		}
	}
}

func TestSettingsRemoveCase1(t *testing.T) {
	l := newSettingsSkipList()
	e1 := &settingsElement{key: 7}
	e2 := &settingsElement{key: 6}
	l.zero.next[0] = e1
	l.zero.next[1] = e1
	e1.next[0] = e2
	e1.height = 1

	l.Remove(6)
}

func TestSettingsRemoveAfterCase1(t *testing.T) {
	l := newSettingsSkipList()
	e1 := &settingsElement{key: 7}
	e2 := &settingsElement{key: 6}
	e3 := &settingsElement{key: 5}
	e4 := &settingsElement{key: 4}
	l.zero.next[0] = e1
	l.zero.next[1] = e1
	e1.next[0] = e2
	e1.height = 1
	e2.next[0] = e3
	e2.height = 1
	e3.next[0] = e4
	e3.height = 1

	l.RemoveAfter(6)
}

func TestSettingsFurtherLessCoverageUp(t *testing.T) {
	l := newSettingsSkipList()
	par := &l.zero
	for _, tc := range []struct {
		val    pt.ID
		height int
	}{
		{val: 6, height: 6},
		{val: 5, height: 6},
		{val: 4, height: 4},
	} {
		cur := newSLSettingsElement(tc.val, tc.height)
		for i := 0; i < cur.Height(); i++ {
			par.setNextLevel(i, cur)
		}
		par = cur
	}

	//	t.Logf("\n%v", l.Dump())

	l.Get(9)
	l.Remove(3)
}
