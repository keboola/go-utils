// Package testproject implements locking of Keboola Projects for E2E parallel tests.
//
// Project is locked:
// - at the host level (flock.Flock)
// - at the goroutines level (sync.Mutex)
//
// Only one test can access the project at a time. See GetTestProject function.
// If there is no unlocked project, the function waits until a project is released.
// Project lock is automatically released at the end of the test.
//
// Package can be safely used in parallel tests that run on a single host.
// Locking between multiple hosts is not provided.
//
// The state of the project does not change automatically,
// if you need an empty project use storageapi.CleanProjectRequest.
package testproject

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

var projects []*Project      // nolint gochecknoglobals
var initLock = &sync.Mutex{} // nolint gochecknoglobals

// Project represents a testing project for E2E tests.
type Project struct {
	t               *testing.T
	storageAPIHost  string
	storageAPIToken string
	projectID       int

	fsLock *flock.Flock // fsLock between processes
	lock   *sync.Mutex  // lock between goroutines
	locked bool
}

// GetTestProject locks and returns a testing project specified in TEST_KBC_PROJECTS environment variable.
// Project lock is automatically released at the end of the test.
// If no project is available, the function waits until a project is released.
func GetTestProject(t *testing.T) *Project {
	t.Helper()
	initProjects()

	if len(projects) == 0 {
		panic(fmt.Errorf(`no test project`))
	}

	for {
		// Try to find a free project
		for _, p := range projects {
			if p.tryLock(t) {
				return p
			}
		}

		// No free project -> wait
		time.Sleep(100 * time.Millisecond)
	}
}

// ID returns id of the project.
func (p *Project) ID() int {
	p.assertLocked()
	return p.projectID
}

// StorageAPIHost returns Storage API host of the project stack.
func (p *Project) StorageAPIHost() string {
	p.assertLocked()
	return p.storageAPIHost
}

// StorageAPIToken returns Storage API token of the project.
func (p *Project) StorageAPIToken() string {
	p.assertLocked()
	return p.storageAPIToken
}

func (p *Project) assertLocked() {
	if !p.locked {
		panic(fmt.Errorf(`test project "%d" is not locked`, p.projectID))
	}
}

func (p *Project) tryLock(t *testing.T) bool {
	// This FS lock works between processes
	if locked, err := p.fsLock.TryLock(); err != nil {
		panic(fmt.Errorf(`cannot lock test project: %w`, err))
	} else if !locked {
		// Busy
		return false
	}

	// This lock works inside one process, between goroutines
	if !p.lock.TryLock() {
		// Busy
		return false
	}

	// Locked
	p.t = t
	p.locked = true

	// Unlock, when test is done
	p.t.Cleanup(func() {
		p.unlock()
	})

	return true
}

// unlock project if it is no more needed in test.
func (p *Project) unlock() {
	defer p.lock.Unlock()
	p.locked = false
	p.t = nil
	if err := p.fsLock.Unlock(); err != nil {
		panic(fmt.Errorf(`cannot unlock test project: %w`, err))
	}
}

// newProject - create test project handler and lock it.
func newProject(host string, id int, token string) *Project {
	// Create locks dir if not exists
	locksDir := filepath.Join(os.TempDir(), `.keboola-as-code-locks`)
	if err := os.MkdirAll(locksDir, 0o700); err != nil {
		panic(fmt.Errorf(`cannot lock test project: %s`, err))
	}

	// lock file name
	lockFile := host + `-` + strconv.Itoa(id) + `.lock`
	lockPath := filepath.Join(locksDir, lockFile)

	return &Project{storageAPIHost: host, projectID: id, storageAPIToken: token, lock: &sync.Mutex{}, fsLock: flock.New(lockPath)}
}

func initProjects() {
	initLock.Lock()
	defer initLock.Unlock()

	// Init only once
	if projects != nil {
		return
	}

	// Multiple test projects
	if def, found := os.LookupEnv(`TEST_KBC_PROJECTS`); found {
		// Each project definition is separated by ";"
		for _, p := range strings.Split(def, ";") {
			p := strings.TrimSpace(p)
			if len(p) == 0 {
				break
			}

			// Definition format: storage_api_host|project_id|project_token
			parts := strings.Split(p, `|`)

			// Check number of parts
			if len(parts) != 3 {
				panic(fmt.Errorf(
					`project definition in TEST_PROJECTS env must be in "storage_api_host|project_id|project_token " format, given "%s"`,
					p,
				))
			}

			host := strings.TrimSpace(parts[0])
			id := strings.TrimSpace(parts[1])
			token := strings.TrimSpace(parts[2])
			idInt, err := strconv.Atoi(id)
			if err != nil {
				panic(fmt.Errorf(`project ID = "%s" is not valid integer`, id))
			}
			projects = append(projects, newProject(host, idInt, token))
		}
	}

	// No test project
	if len(projects) == 0 {
		panic(fmt.Errorf(`please specify one or more Keboola Connection testing projects by TEST_KBC_PROJECTS env, in format "<storage_api_host>|<project_id>|<token>;..."`))
	}
}
