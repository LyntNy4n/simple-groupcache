package simplegroupcache

import (
	"fmt"
	"log"
	"testing"
)

func TestGet(t *testing.T) {
	db := map[string]string{
		"Tom":  "630",
		"Jack": "589",
		"Sam":  "567",
	}
	cacheStrategy := "lru"
	loadCounts := make(map[string]int, len(db))

	g := NewGroup("scores", 2<<10, cacheStrategy, RetrieverFunc(
		func(key string) ([]byte, error) {
			log.Println("[Mysql] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key]++
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		if view, err := g.Get(k); err != nil || view.String() != v {
			t.Fatalf("failed to get value of %s", k)
		}
		if _, err := g.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}

	if view, err := g.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	} else {
		log.Println(err)
	}
}
