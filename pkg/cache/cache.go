// pkg/cache/cache.go
package cache

import (
	"sync"
	"time"
)

type Item struct {
	Value      interface{}
	Expiration int64
}

type Cache struct {
	items map[string]Item
	mu    sync.RWMutex
}

func NewCache() *Cache {
	cache := &Cache{
		items: make(map[string]Item),
	}
	go cache.startGC()
	return cache
}

func (c *Cache) Set(key string, value interface{}, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiration := time.Now().Add(duration).UnixNano()
	c.items[key] = Item{
		Value:      value,
		Expiration: expiration,
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	if time.Now().UnixNano() > item.Expiration {
		return nil, false
	}

	return item.Value, true
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *Cache) startGC() {
	ticker := time.NewTicker(time.Minute)
	for {
		<-ticker.C
		c.mu.Lock()
		for k, v := range c.items {
			if time.Now().UnixNano() > v.Expiration {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
