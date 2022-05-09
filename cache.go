package gopherlink

import (
	"fmt"
	"sync"

	"github.com/damaredayo/goutubedl"
)

var (
	ErrNoNextSong = fmt.Errorf("no next song")
)

type Cache struct {
	sync.Mutex
	aacCache  map[string][]byte
	infoCache map[string]*goutubedl.Info
	nextAac   []byte
	nextInfo  *goutubedl.Info
}

func (c *Cache) AddSong(url string, aac []byte) {
	c.Lock()
	defer c.Unlock()
	c.aacCache[url] = aac
}

func (c *Cache) GetSong(url string) (aac []byte, ok bool) {
	c.Lock()
	defer c.Unlock()
	aac, ok = c.aacCache[url]
	return
}

func (c *Cache) AddInfo(url string, info *goutubedl.Info) {
	c.Lock()
	defer c.Unlock()
	c.infoCache[url] = info
}

func (c *Cache) GetInfo(url string) (info *goutubedl.Info, ok bool) {
	c.Lock()
	defer c.Unlock()
	info, ok = c.infoCache[url]
	return
}

// Funcs for preloading next song

func (c *Cache) PreloadSong(url string) error {
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

func (c *Cache) GetNextSong() (aac []byte, info *goutubedl.Info, err error) {
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

func (v *VoiceConnection) MakeCache() {
	v.Cache = &Cache{
		aacCache:  make(map[string][]byte),
		infoCache: make(map[string]*goutubedl.Info),
	}
}
