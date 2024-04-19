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

// redisLocker is implementation of locker in which the mutual exclusion of project access is done by locking redis unique ID.
type redisLocker struct {
	projectID   string
	redisClient *redis.Client
	locker      *redislock.Client
	redisLock   *redislock.Lock // lock between projects using redis
	locked      bool
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

func (rl *redisLocker) newForProject(p *Project) locker {
	return &redisLocker{
		projectID:   fmt.Sprintf("%s-%d", p.definition.Host, p.definition.ProjectID),
		redisClient: rl.redisClient,
		locker:      rl.locker,
	}
}

func (rl *redisLocker) tryLock() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	lock, err := rl.locker.Obtain(ctx, rl.projectID, 60*time.Second, nil)
	if errors.Is(err, redislock.ErrNotObtained) {
		return false
	} else if err != nil {
		panic(fmt.Errorf(`cannot lock test project using redis lock: %w`, err))
	}

	rl.redisLock = lock
	rl.locked = true
	return true
}

func (rl *redisLocker) unlock() {
	rl.locked = false
	if err := rl.redisLock.Release(context.Background()); err != nil {
		panic(fmt.Errorf(`cannot unlock test project using redis lock: %w`, err))
	}
}

func (rl *redisLocker) isLocked() bool {
	return rl.locked
}
