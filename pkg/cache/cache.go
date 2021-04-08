package cache

import (
	"runtime/debug"
	"time"

	"github.com/coocood/freecache"
)

type MessageCache struct {
	Cache     *freecache.Cache
	Expire    int
	CacheSize int
}

func NewMessageCache(cacheSizeMB int, secondsToExpire int) *MessageCache {
	cacheSize := cacheSizeMB * 1024 * 1024
	debug.SetGCPercent(20)
	return &MessageCache{
		Cache:     freecache.NewCache(cacheSize),
		Expire:    secondsToExpire,
		CacheSize: cacheSize,
	}
}

func (qc *MessageCache) Push(v string) error {
	key := time.Now().UnixNano()
	value := []byte(v)

	err := qc.Cache.SetInt(key, value, qc.Expire)
	return err
}
