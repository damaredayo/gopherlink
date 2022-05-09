package gopherlink

import (
	"context"
	"fmt"
	"sync"

	pb "github.com/damaredayo/gopherlink/proto"
	"github.com/go-redis/redis/v8"
)

type QueueTypeConst int

const (
	RedisQueueType QueueTypeConst = iota
	InternalQueueType
)

var (
	ErrNoSongFound  = fmt.Errorf("no song found")
	IndexNotInRange = fmt.Errorf("index not in range")
)

func (q QueueTypeConst) String() string {
	switch q {
	case RedisQueueType:
		return "redis"
	case InternalQueueType:
		return "internal"
	default:
		return "unknown"
	}
}

type Queue struct {
	QueueType
}

type QueueType interface {
	AddSong(ctx context.Context, song *pb.SongInfo) error
	RemoveSong(ctx context.Context, song *pb.SongInfo) error
	RemoveSongByIndex(ctx context.Context, index int) (*pb.SongInfo, error)
	GetQueue(ctx context.Context) ([]*pb.SongInfo, error)
	GetQueueType() QueueTypeConst
	GetNextSong(ctx context.Context) (*pb.SongInfo, error)
	Ready() bool
}

type RedisQueue struct {
	ready bool
	redis *redis.Client
}

type InternalQueue struct {
	sync.Mutex
	queue []*pb.SongInfo
}

func (v *VoiceConnection) MakeQueue(qtype QueueTypeConst, options interface{}) error {
	switch qtype {
	case RedisQueueType:
		opts, ok := options.(*redis.Options)
		if !ok {
			return fmt.Errorf("invalid options")
		}

		redisStruct := &RedisQueue{redis: redis.NewClient(opts)}

		if opts.OnConnect == nil {
			opts.OnConnect = redisStruct.readyCallback
		} else {
			// preserve the users ready callback as well as add ours - futureproofing
			current := opts.OnConnect
			opts.OnConnect = func(ctx context.Context, conn *redis.Conn) error {
				err := current(ctx, conn)
				if err != nil {
					return err
				}
				return redisStruct.readyCallback(ctx, conn)
			}
		}

		redisStruct.redis.Options().OnConnect = redisStruct.readyCallback
		v.SetQueueType(redisStruct)

	case InternalQueueType:
		v.SetQueueType(&InternalQueue{
			queue: make([]*pb.SongInfo, 0),
		})
	}

	return nil
}

func (v *VoiceConnection) SetQueueType(q QueueType) {
	v.Queue = &Queue{q}
}

// redis queue

func (r *RedisQueue) AddSong(ctx context.Context, song *pb.SongInfo) error {
	// TODO: add song to redis
	return nil
}

func (r *RedisQueue) RemoveSong(ctx context.Context, song *pb.SongInfo) error {
	// TODO: remove song from redis
	return nil
}

func (r *RedisQueue) RemoveSongByIndex(ctx context.Context, index int) (*pb.SongInfo, error) {
	// TODO: remove song from redis
	return nil, nil
}

func (r *RedisQueue) GetQueue(ctx context.Context) ([]*pb.SongInfo, error) {
	// TODO: get queue from redis
	return nil, nil
}

func (r *RedisQueue) GetNextSong(ctx context.Context) (*pb.SongInfo, error) {
	return nil, nil
}

func (r *RedisQueue) Ready() bool {
	return r.ready
}

func (r *RedisQueue) readyCallback(ctx context.Context, conn *redis.Conn) error {
	if err := r.redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis connection error: %s", err)
	}
	r.ready = true
	return nil
}

func (r *RedisQueue) GetQueueType() QueueTypeConst {
	return RedisQueueType
}

// internal queue

func (i *InternalQueue) AddSong(ctx context.Context, song *pb.SongInfo) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		i.Lock()
		defer i.Unlock()
		i.queue = append(i.queue, song)
		return nil
	}
}

func (i *InternalQueue) RemoveSong(ctx context.Context, song *pb.SongInfo) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		i.Lock()
		defer i.Unlock()
		for index, s := range i.queue {
			if s == song {
				// remove from slice
				i.queue = append(i.queue[:index], i.queue[index+1:]...)
				return nil
			}
		}
		return ErrNoSongFound
	}
}

func (i *InternalQueue) RemoveSongByIndex(ctx context.Context, remove int) (*pb.SongInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		i.Lock()
		defer i.Unlock()
		// remove from slice by index safely
		for index, s := range i.queue {
			if index == remove {
				i.queue = append(i.queue[:index], i.queue[index+1:]...)
				return s, nil
			}
		}
		return nil, IndexNotInRange
	}
}

func (i *InternalQueue) GetQueue(ctx context.Context) ([]*pb.SongInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		i.Lock()
		defer i.Unlock()
		return i.queue, nil
	}
}

func (i *InternalQueue) GetNextSong(ctx context.Context) (*pb.SongInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		i.Lock()
		defer i.Unlock()
		if len(i.queue) == 0 {
			return nil, ErrNoNextSong
		}
		song := i.queue[0]
		if len(i.queue) > 1 {
			i.queue = i.queue[1:]
		} else {
			i.queue = make([]*pb.SongInfo, 0)
		}
		return song, nil
	}
}

func (i *InternalQueue) GetQueueType() QueueTypeConst {
	return InternalQueueType
}

func (i *InternalQueue) Ready() bool {
	if i.queue != nil {
		return true
	}
	return false
}
