package llm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"sort"
	"sync"
)

// Cache — файл ключ→ответ. Коммитится в репо: без ключа пайплайн берёт тексты отсюда и остаётся воспроизводимым.
type Cache struct {
	path  string
	mu    sync.Mutex
	items map[string]string
	dirty bool
}

func OpenCache(path string) *Cache {
	c := &Cache{path: path, items: map[string]string{}}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &c.items)
	}
	return c
}

func Key(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:24]
}

func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.items[key]
	return v, ok
}

func (c *Cache) Put(key, val string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = val
	c.dirty = true
}

func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}

// Save — детерминированный JSON (ключи отсортированы), только если были изменения.
func (c *Cache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dirty {
		return nil
	}
	keys := make([]string, 0, len(c.items))
	for k := range c.items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf []byte
	buf = append(buf, "{\n"...)
	for i, k := range keys {
		kb, _ := json.Marshal(k)
		vb, _ := json.Marshal(c.items[k])
		buf = append(buf, "  "...)
		buf = append(buf, kb...)
		buf = append(buf, ": "...)
		buf = append(buf, vb...)
		if i < len(keys)-1 {
			buf = append(buf, ',')
		}
		buf = append(buf, '\n')
	}
	buf = append(buf, "}\n"...)
	if err := os.WriteFile(c.path, buf, 0o644); err != nil {
		return err
	}
	c.dirty = false
	return nil
}
