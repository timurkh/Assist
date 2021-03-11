package db

import "sync"

type userDataCache struct {
	mx sync.RWMutex
	m  map[string]UserData
}

func NewUserDataCache() *userDataCache {
	return &userDataCache{
		m: make(map[string]UserData),
	}
}

func (c *userDataCache) Get(id string) (UserData, bool) {
	c.mx.RLock()
	val, ok := c.m[id]
	c.mx.RUnlock()
	return val, ok
}

func (c *userDataCache) Set(key string, value UserData) {
	c.mx.Lock()
	c.m[key] = value
	c.mx.Unlock()
}

//TODO: set max size of the cache
