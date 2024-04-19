package testproject

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gofrs/flock"
)

// fsLocker is implementation of locker using flock and mutex to perform mutual exclusion of project access on both process and goroutine level.
type fsLocker struct {
	projectID string
	locksDir  string
	lock      *sync.Mutex  // lock between goroutines
	fsLock    *flock.Flock // fsLock between processes
	locked    bool
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

func (fl *fsLocker) newForProject(p *Project) locker {
	// Get lock file name
	projectID := p.definition.Host + `-` + strconv.Itoa(p.definition.ProjectID) + `.lock`
	lockPath := filepath.Join(fl.locksDir, projectID)
	fsLock := flock.New(lockPath)
	return &fsLocker{
		projectID: projectID,
		locksDir:  fl.locksDir,
		lock:      &sync.Mutex{},
		fsLock:    fsLock,
	}
}

func (fl *fsLocker) tryLock() bool {
	// This FS lock works between processes
	if locked, err := fl.fsLock.TryLock(); err != nil {
		panic(fmt.Errorf(`cannot lock test project: %w`, err))
	} else if !locked {
		// Busy
		return false
	}

	// This lock works inside one process, between goroutines
	if !fl.lock.TryLock() {
		// Busy
		return false
	}

	// Locked
	fl.locked = true
	return true
}

// unlock project if it is no more needed in test.
func (fl *fsLocker) unlock() {
	defer fl.lock.Unlock()
	fl.locked = false
	if err := fl.fsLock.Unlock(); err != nil {
		panic(fmt.Errorf(`cannot unlock test project: %w`, err))
	}
}

func (fl *fsLocker) isLocked() bool {
	return fl.locked
}
