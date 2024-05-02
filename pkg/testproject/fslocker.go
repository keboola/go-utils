package testproject

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gofrs/flock"
)

// fsLocker is factory constructing fsProjectLockers.
type fsLocker struct {
	locksDir string
}

func newFsLocker() (*fsLocker, error) {
	// Get locks dir name
	lockDirName, found := os.LookupEnv(TestKbcProjectsLockDirNameKey)
	if !found {
		// Default value
		lockDirName = ".keboola-as-code-locks"
	}

	// Create locks dir if not exists
	locksDir := filepath.Join(os.TempDir(), lockDirName)
	if err := os.MkdirAll(locksDir, 0o700); err != nil {
		return nil, fmt.Errorf(`cannot create locks dir: %w`, err)
	}

	return &fsLocker{
		locksDir: locksDir,
	}, nil
}

// fsProjectLocker is implementation of locker using flock and mutex to perform mutual exclusion of project access on both process and goroutine level.
type fsProjectLocker struct {
	fsLocker  *fsLocker
	projectID string
	lock      *sync.Mutex  // lock between goroutines
	fsLock    *flock.Flock // fsLock between processes
	locked    bool
}

func (fl *fsLocker) newForProject(p *Project) projectLocker {
	// Get lock file name
	projectID := p.definition.Host + `-` + strconv.Itoa(p.definition.ProjectID) + `.lock`
	lockPath := filepath.Join(fl.locksDir, projectID)
	fsLock := flock.New(lockPath)
	return &fsProjectLocker{
		fsLocker:  fl,
		projectID: projectID,
		lock:      &sync.Mutex{},
		fsLock:    fsLock,
	}
}

func (fl *fsProjectLocker) tryLock(ctx context.Context) (bool, context.CancelFunc) {
	// This FS lock works between processes
	if locked, err := fl.fsLock.TryLock(); err != nil {
		panic(fmt.Errorf(`cannot lock test project: %w`, err))
	} else if !locked {
		// Busy
		return false, nil
	}

	// This lock works inside one process, between goroutines
	if !fl.lock.TryLock() {
		// Busy
		return false, nil
	}

	cancel := func() { fl.unlock() }
	// Locked
	fl.locked = true
	return true, cancel
}

// unlock project if it is no more needed in test.
func (fl *fsProjectLocker) unlock() {
	defer fl.lock.Unlock()
	fl.locked = false
	if err := fl.fsLock.Unlock(); err != nil {
		panic(fmt.Errorf(`cannot unlock test project: %w`, err))
	}
}

func (fl *fsProjectLocker) isLocked() bool {
	return fl.locked
}
