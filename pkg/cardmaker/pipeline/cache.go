package pipeline

import (
	"encoding/json"
	"os"
	"sync"
)

const cacheFileName = ".cache.json"

// cache remembers the key each output was last built with. A missing or
// corrupt file is treated as an empty cache: worst case, everything is
// rebuilt.
type cache struct {
	mu   sync.Mutex
	keys map[string]string
}

func loadCache(path string) *cache {
	c := &cache{keys: map[string]string{}}

	data, err := os.ReadFile(path)
	if err != nil {
		return c
	}
	if err := json.Unmarshal(data, &c.keys); err != nil || c.keys == nil {
		c.keys = map[string]string{}
	}

	return c
}

func (c *cache) matches(name, key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.keys[name] == key
}

func (c *cache) get(name string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, ok := c.keys[name]

	return value, ok
}

func (c *cache) set(name, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.keys[name] = key
}

func (c *cache) save(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.MarshalIndent(c.keys, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(path, append(data, '\n'))
}
