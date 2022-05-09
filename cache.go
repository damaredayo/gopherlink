package gopherlink

import (
	"context"
	"fmt"
	"sync"

	"github.com/damaredayo/goutubedl"
	"github.com/go-redis/redis/v8"
)

type CacheTypeConst int

var (
	ErrNoNextSong = fmt.Errorf("no next song")
)

const (
	CacheTypeInternal CacheTypeConst = iota
	CacheTypeRedis
)

type internalCache struct {
	sync.Mutex
	aacCache  map[string][]byte
	infoCache map[string]*goutubedl.Info
	nextAac   []byte
	nextInfo  *goutubedl.Info
}

type redisCache struct {
	sync.Mutex
	ready    bool
	redis    *redis.Client
	aacCache map[string][]byte
	nextAac  []byte
	nextInfo *goutubedl.Info
}

type Cache interface {
	AddSong(url string, aac []byte)
	GetSong(url string) (aac []byte, ok bool)
	AddInfo(url string, info *goutubedl.Info)
	GetInfo(url string) (info *goutubedl.Info, ok bool)
	// The following are internal no matter what
	PreloadSong(url string) error
	GetNextSong() (aac []byte, info *goutubedl.Info, err error)
}

func (v *VoiceConnection) MakeCache(c CacheTypeConst, options interface{}) error {
	switch c {
	case CacheTypeInternal:
		v.Cache = &internalCache{
			aacCache:  make(map[string][]byte),
			infoCache: make(map[string]*goutubedl.Info),
		}
	case CacheTypeRedis:
		opts, ok := options.(*redis.Options)
		if !ok {
			return fmt.Errorf("invalid options")
		}

		redisCache := &redisCache{redis: redis.NewClient(opts)}

		if opts.OnConnect == nil {
			opts.OnConnect = redisCache.readyCallback
		} else {
			// preserve the users ready callback as well as add ours - futureproofing
			current := opts.OnConnect
			opts.OnConnect = func(ctx context.Context, conn *redis.Conn) error {
				err := current(ctx, conn)
				if err != nil {
					return err
				}
				return redisCache.readyCallback(ctx, conn)
			}
		}

		redisCache.redis.Options().OnConnect = redisCache.readyCallback
		v.Cache = redisCache

	}
	return nil
}

// Internal cache

func (c *internalCache) AddSong(url string, aac []byte) {
	c.Lock()
	defer c.Unlock()
	c.aacCache[url] = aac
}

func (c *internalCache) GetSong(url string) (aac []byte, ok bool) {
	c.Lock()
	defer c.Unlock()
	aac, ok = c.aacCache[url]
	return
}

func (c *internalCache) AddInfo(url string, info *goutubedl.Info) {
	c.Lock()
	defer c.Unlock()
	c.infoCache[url] = info
}

func (c *internalCache) GetInfo(url string) (info *goutubedl.Info, ok bool) {
	c.Lock()
	defer c.Unlock()
	info, ok = c.infoCache[url]
	return
}

func (c *internalCache) PreloadSong(url string) error {
	c.Lock()
	defer c.Unlock()
	// if theres already something in the cache, don't preload
	if c.nextAac != nil || c.nextInfo != nil {
		return nil
	}

	if _, ok := c.aacCache[url]; !ok {
		aac, info, err := YoutubeToAAC(url)
		if err != nil {
			return err
		}
		c.aacCache[url], c.nextAac = aac, aac
		c.infoCache[url], c.nextInfo = info, info
	}
	return nil
}

func (c *internalCache) GetNextSong() (aac []byte, info *goutubedl.Info, err error) {
	c.Lock()
	defer c.Unlock()
	aac, info = c.nextAac, c.nextInfo
	if aac == nil || info == nil {
		err = ErrNoNextSong
		return
	}
	c.nextAac, c.nextInfo = nil, nil
	return
}

// Redis cache

func (r *redisCache) AddSong(url string, aac []byte) {
	return
}

func (r *redisCache) GetSong(url string) (aac []byte, ok bool) {
	return
}

func (r *redisCache) AddInfo(url string, info *goutubedl.Info) {
	return
}

func (r *redisCache) GetInfo(url string) (info *goutubedl.Info, ok bool) {
	return
}

func (r *redisCache) PreloadSong(url string) error {
	r.Lock()
	defer r.Unlock()
	// if theres already something in the cache, don't preload
	if r.nextAac != nil || r.nextInfo != nil {
		return nil
	}
	if _, ok := r.aacCache[url]; !ok {
		aac, info, err := YoutubeToAAC(url)
		if err != nil {
			return err
		}
		r.nextAac = aac
		r.nextInfo = info
	}
	return nil
}

func (r *redisCache) GetNextSong() (aac []byte, info *goutubedl.Info, err error) {
	r.Lock()
	defer r.Unlock()
	aac, info = r.nextAac, r.nextInfo
	if aac == nil || info == nil {
		err = ErrNoNextSong
		return
	}
	r.nextAac, r.nextInfo = nil, nil
	return
}

func (r *redisCache) readyCallback(ctx context.Context, conn *redis.Conn) error {
	if err := r.redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis connection error: %s", err)
	}
	r.ready = true
	return nil
}
