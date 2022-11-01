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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslation "github.com/go-playground/validator/v10/translations/en"
	"github.com/gofrs/flock"
)

var projects []*Project      // nolint gochecknoglobals
var initLock = &sync.Mutex{} // nolint gochecknoglobals

// Project represents a testing project for E2E tests.
type Project struct {
	Host           string `json:"host" validate:"required"`
	Token          string `json:"token" validate:"required"`
	StagingStorage string `json:"stagingStorage" validate:"required"`
	ProjectID      int    `json:"project" validate:"required"`

	fsLock *flock.Flock `json:"-"` // fsLock between processes
	lock   *sync.Mutex  `json:"-"` // lock between goroutines
	locked bool         `json:"-"`
}

type UnlockFn func()

// GetTestProjectForTest locks and returns a testing project specified in TEST_KBC_PROJECTS environment variable.
// Project lock is automatically released at the end of the test.
// If no project is available, the function waits until a project is released.
func GetTestProjectForTest(t *testing.T, opts ...Option) (*Project, error) {
	t.Helper()

	// Get project
	p, unlockFn, err := GetTestProject(opts...)
	if err != nil {
		return nil, err
	}

	// Unlock when test is done
	t.Cleanup(func() {
		unlockFn()
	})

	return p, nil
}

type Option func(c *config)

type config struct {
	stagingStorage string
}

func WithStagingStorage(stagingStorage string) Option {
	return func(c *config) {
		c.stagingStorage = stagingStorage
	}
}

// GetTestProject locks and returns a testing project specified in TEST_KBC_PROJECTS environment variable.
// The returned UnlockFn function must be called to free project, when the project is no longer used (e.g. defer unlockFn())
// If no project is available, the function waits until a project is released.
func GetTestProject(opts ...Option) (*Project, UnlockFn, error) {
	c := &config{}
	for _, opt := range opts {
		opt(c)
	}

	err := initProjects()
	if err != nil {
		return nil, nil, err
	}

	if len(projects) == 0 {
		return nil, nil, fmt.Errorf(`no test project`)
	}

	for {
		// Try to find a free project
		anyProjectFound := false
		for _, p := range projects {
			if c.stagingStorage == "" || p.StagingStorage == c.stagingStorage {
				if p.tryLock() {
					return p, func() {
						p.unlock()
					}, nil
				}
				anyProjectFound = true
			}
		}

		if !anyProjectFound {
			return nil, nil, fmt.Errorf(fmt.Sprintf(`no test project for staging storage %s`, c.stagingStorage))
		}

		// No free project -> wait
		time.Sleep(100 * time.Millisecond)
	}
}

// ID returns id of the project.
func (p *Project) ID() int {
	p.assertLocked()
	return p.ProjectID
}

// StorageAPIHost returns Storage API host of the project stack.
func (p *Project) StorageAPIHost() string {
	p.assertLocked()
	return p.Host
}

// StorageAPIToken returns Storage API token of the project.
func (p *Project) StorageAPIToken() string {
	p.assertLocked()
	return p.Token
}

func (p *Project) assertLocked() {
	if !p.locked {
		panic(fmt.Errorf(`test project "%d" is not locked`, p.ProjectID))
	}
}

func (p *Project) tryLock() bool {
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
	p.locked = true
	return true
}

// unlock project if it is no more needed in test.
func (p *Project) unlock() {
	defer p.lock.Unlock()
	p.locked = false
	if err := p.fsLock.Unlock(); err != nil {
		panic(fmt.Errorf(`cannot unlock test project: %w`, err))
	}
}

// initProject - init test project handler and lock it.
func initProject(project *Project, validate *validator.Validate) error {
	err := validate.Struct(project)
	if err != nil {
		return err
	}

	// Get locks dir name
	lockDirName, found := os.LookupEnv("TEST_KBC_PROJECTS_LOCK_DIR_NAME")
	if !found {
		// Default value
		lockDirName = ".keboola-as-code-locks"
	}

	// Create locks dir if not exists
	locksDir := filepath.Join(os.TempDir(), lockDirName)
	if err := os.MkdirAll(locksDir, 0o700); err != nil {
		panic(fmt.Errorf(`cannot lock test project: %w`, err))
	}

	// lock file name
	lockFile := project.Host + `-` + strconv.Itoa(project.ProjectID) + `.lock`
	lockPath := filepath.Join(locksDir, lockFile)

	project.lock = &sync.Mutex{}
	project.fsLock = flock.New(lockPath)

	return nil
}

func resetProjects() {
	initLock.Lock()
	defer initLock.Unlock()
	projects = nil
}

func initProjects() error {
	initLock.Lock()
	defer initLock.Unlock()

	// Init only once
	if projects != nil {
		return nil
	}

	projects = make([]*Project, 0)
	if def, found := os.LookupEnv(`TEST_KBC_PROJECTS`); found {
		if def == "" {
			return fmt.Errorf(`please specify one or more Keboola Connection testing projects by TEST_KBC_PROJECTS env, in format '[{"host":"","token":"","project":"","stagingStorage":""}]'`)
		}
		err := json.Unmarshal([]byte(def), &projects)
		if err != nil {
			return fmt.Errorf(`decoding of env var TEST_KBC_PROJECTS failed: %w`, err)
		}
	}

	// No test project
	if len(projects) == 0 {
		return fmt.Errorf(`please specify one or more Keboola Connection testing projects by TEST_KBC_PROJECTS env, in format '[{"host":"","token":"","project":"","stagingStorage":""}]'`)
	}

	validate := validator.New()
	translator := ut.New(en.New()).GetFallback()
	if err := enTranslation.RegisterDefaultTranslations(validate, translator); err != nil {
		panic(err)
	}
	for _, p := range projects {
		err := initProject(p, validate)
		if err != nil {
			return fmt.Errorf(`initialization of project %d failed: %w`, p.ProjectID, err)
		}
	}

	return nil
}
