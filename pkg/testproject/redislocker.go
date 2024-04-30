package testproject

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bsm/redislock"
	"github.com/redis/go-redis/v9"
)

const (
	TTL = 60 * time.Second
)

// redisLocker is factory constructing redisProjectLockers.
type redisLocker struct {
	redisClient *redis.Client
	locker      *redislock.Client
}

func newRedisLocker(redisHost, redisPassword string) (*redisLocker, error) {
	var client *redis.Client
	var locker *redislock.Client
	_, after, found := strings.Cut(redisHost, "://")
	if !found {
		return nil, errors.New("no protocol specified")
	}

	opts := &redis.Options{
		Addr:     after,
		Password: redisPassword,
	}
	if withTls := strings.Contains(redisHost, "+tls"); withTls {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	client = redis.NewClient(opts)
	result := client.Ping(context.Background())
	if result.Err() != nil {
		client.Close()
		return nil, result.Err()
	}

	locker = redislock.New(client)
	return &redisLocker{
		redisClient: client,
		locker:      locker,
	}, nil
}

// redisProjectLocker is implementation of locker in which the mutual exclusion of project access is done by locking redis unique ID.
type redisProjectLocker struct {
	redisLocker *redisLocker
	projectID   string
	redisLock   *redislock.Lock // lock between projects using redis
	locked      bool
}

func (rl *redisLocker) newForProject(p *Project) projectLocker {
	return &redisProjectLocker{
		redisLocker: rl,
		projectID:   fmt.Sprintf("%s-%d", p.definition.Host, p.definition.ProjectID),
	}
}

func (rl *redisProjectLocker) tryLock() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	lock, err := rl.redisLocker.locker.Obtain(ctx, rl.projectID, TTL, nil)
	if errors.Is(err, redislock.ErrNotObtained) {
		return false
	} else if err != nil {
		panic(fmt.Errorf(`cannot lock test project using redis lock: %w`, err))
	}

	rl.redisLock = lock
	go rl.extendLock()
	rl.locked = true
	return true
}

// extendLock extends the lock forewer when TTL/2 passed.
// replace implementation with https://github.com/bsm/redislock/pull/73 in future.
func (rl *redisProjectLocker) extendLock() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(TTL / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			err := rl.redisLock.Refresh(ctx, TTL, nil)
			if err != nil {
				if errors.Is(err, redislock.ErrNotObtained) {
					return
				}

				panic(fmt.Errorf(`cannot extend the redis lock: %w`, err))
			}
		}
	}
}

func (rl *redisProjectLocker) unlock() {
	rl.locked = false
	if err := rl.redisLock.Release(context.Background()); err != nil {
		panic(fmt.Errorf(`cannot unlock test project using redis lock: %w`, err))
	}
}

func (rl *redisProjectLocker) isLocked() bool {
	return rl.locked
}
