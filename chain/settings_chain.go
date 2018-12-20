package chain

import (
	"sync"

	"github.com/qiwitech/qdp/pt"
)

type SettingsChain struct {
	mu   sync.Mutex
	list map[pt.AccID]*settingsSkiplist
}

func NewSettingsChain() *SettingsChain {
	return &SettingsChain{
		list: make(map[pt.AccID]*settingsSkiplist),
	}
}

func (c *SettingsChain) Put(s *pt.Settings) {
	if s.ID == 0 {
		panic("settings chain put: zero id")
	}

	defer c.mu.Unlock()
	c.mu.Lock()

	list, ok := c.list[s.Account]
	if !ok {
		list = newSettingsSkipList()
		c.list[s.Account] = list
	}

	// cut chain
	if e := list.Front(); e != nil {
		id := e.Settings.ID

		if id > 1 {
			list.RemoveAfter(id - 1)
		}
	}

	e := list.GetOrInsert(s.ID)
	if e == nil {
		panic("can't insert element")
	}
	if e.Settings == nil {
		e.Settings = s
	}
}

func (c *SettingsChain) GetLastSettings(accID pt.AccID) *pt.Settings {
	defer c.mu.Unlock()
	c.mu.Lock()

	if list, ok := c.list[accID]; ok {
		if e := list.Front(); e != nil {
			return e.Settings
		}
	}

	return nil
}

func (c *SettingsChain) GetLastHash(accID pt.AccID) pt.Hash {
	if s := c.GetLastSettings(accID); s != nil {
		return s.Hash
	}
	return pt.ZeroHash
}

func (c *SettingsChain) Reset(accID pt.AccID) {
	defer c.mu.Unlock()
	c.mu.Lock()

	delete(c.list, accID)
}
