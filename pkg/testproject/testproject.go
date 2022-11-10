// Package testproject implements locking of Keboola Projects for E2E parallel tests.
//
// Project is locked:
// - at the host level (flock.Flock)
// - at the goroutines level (sync.Mutex)
//
// Only one test can access the project at a time. See GetTestProject function.
// If there is no unlocked project, the function waits until a project is released.
//
// Package can be safely used in parallel tests that run on a single host.
// Use GetTestProjectForTest function to get a testing project in a test.
// Project lock is automatically released at the end of the test.
//
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
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslation "github.com/go-playground/validator/v10/translations/en"
	"github.com/gofrs/flock"
)

const (
	StagingStorageABS = "abs"
	StagingStorageGCS = "gcs"
	StagingStorageS3  = "s3"
)

var pool ProjectsPool        // nolint gochecknoglobals
var poolLock = &sync.Mutex{} // nolint gochecknoglobals

// ProjectsPool a group of testing projects.
type ProjectsPool []*Project

// Project represents a testing project for E2E tests.
type Project struct {
	definition Definition
	fsLock     *flock.Flock // fsLock between processes
	lock       *sync.Mutex  // lock between goroutines
	locked     bool
}

// Definition is project Definition parsed from the ENV.
type Definition struct {
	Host           string `json:"host" validate:"required"`
	Token          string `json:"token" validate:"required"`
	StagingStorage string `json:"stagingStorage" validate:"required"`
	ProjectID      int    `json:"project" validate:"required"`
}

// UnlockFn must be called if the project is no longer used.
type UnlockFn func()

// Option for the GetTestProjectForTest and GetTestProject functions.
type Option func(c *config)

// config for the GetTestProjectForTest and GetTestProject functions.
type config struct {
	stagingStorage string
}

// TInterface is cleanup part of the *testing.T.
type TInterface interface {
	Cleanup(f func())
}

func WithStagingStorageABS() Option {
	return func(c *config) {
		c.stagingStorage = StagingStorageABS
	}
}

func WithStagingStorageGCS() Option {
	return func(c *config) {
		c.stagingStorage = StagingStorageGCS
	}
}

func WithStagingStorageS3() Option {
	return func(c *config) {
		c.stagingStorage = StagingStorageS3
	}
}

func WithStagingStorage(stagingStorage string) Option {
	return func(c *config) {
		c.stagingStorage = stagingStorage
	}
}

// GetTestProjectForTest locks and returns a testing project specified in TEST_KBC_PROJECTS environment variable.
// Project lock is automatically released at the end of the test.
// If no project is available, the function waits until a project is released.
func GetTestProjectForTest(t TInterface, opts ...Option) (*Project, error) {
	return mustGetProjects().GetTestProjectForTest(t, opts...)
}

// GetTestProject locks and returns a testing project specified in TEST_KBC_PROJECTS environment variable.
// The returned UnlockFn function must be called to free project, when the project is no longer used (e.g. defer unlockFn())
// If no project is available, the function waits until a project is released.
func GetTestProject(opts ...Option) (*Project, UnlockFn, error) {
	return mustGetProjects().GetTestProject(opts...)
}

// GetTestProjectForTest locks and returns a testing project specified in TEST_KBC_PROJECTS environment variable.
// Project lock is automatically released at the end of the test.
// If no project is available, the function waits until a project is released.
func (v ProjectsPool) GetTestProjectForTest(t TInterface, opts ...Option) (*Project, error) {
	// Get project
	p, unlockFn, err := v.GetTestProject(opts...)
	if err != nil {
		return nil, err
	}

	// Unlock when test is done
	t.Cleanup(func() {
		unlockFn()
	})

	return p, nil
}

// GetTestProject locks and returns a testing project specified in TEST_KBC_PROJECTS environment variable.
// The returned UnlockFn function must be called to free project, when the project is no longer used (e.g. defer unlockFn())
// If no project is available, the function waits until a project is released.
func (v ProjectsPool) GetTestProject(opts ...Option) (*Project, UnlockFn, error) {
	c := &config{}
	for _, opt := range opts {
		opt(c)
	}

	if len(v) == 0 {
		return nil, nil, fmt.Errorf(`no test project`)
	}

	for {
		// Try to find a free project
		anyProjectFound := false
		for _, p := range v {
			if c.stagingStorage == "" || p.definition.StagingStorage == c.stagingStorage {
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
	return p.definition.ProjectID
}

// StorageAPIHost returns Storage API host of the project stack.
func (p *Project) StorageAPIHost() string {
	p.assertLocked()
	return p.definition.Host
}

// StorageAPIToken returns Storage API token of the project.
func (p *Project) StorageAPIToken() string {
	p.assertLocked()
	return p.definition.Token
}

// StagingStorage returns staging storage of the project Definition.
func (p *Project) StagingStorage() string {
	p.assertLocked()
	return p.definition.StagingStorage
}

func (p *Project) assertLocked() {
	if !p.locked {
		panic(fmt.Errorf(`test project "%d" is not locked`, p.definition.ProjectID))
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

func MustGetProjectsFrom(str string) ProjectsPool {
	projects, err := GetProjectsFrom(str)
	if err != nil {
		panic(err)
	}
	return projects
}

func GetProjectsFrom(str string) (ProjectsPool, error) {
	// No test project
	if str == "" {
		return nil, fmt.Errorf(`please specify one or more Keboola Connection testing projects in format '[{"host":"","token":"","project":"","stagingStorage":""}]'`)
	}

	// Decode the value
	defs := make([]Definition, 0)
	if err := json.Unmarshal([]byte(str), &defs); err != nil {
		return nil, fmt.Errorf(`decoding failed: %w`, err)
	}

	// No test project
	if len(defs) == 0 {
		return nil, fmt.Errorf(`please specify one or more Keboola Connection testing projects in format '[{"host":"","token":"","project":"","stagingStorage":""}]'`)
	}

	// Setup validator
	validate := validator.New()
	translator := ut.New(en.New()).GetFallback()
	if err := enTranslation.RegisterDefaultTranslations(validate, translator); err != nil {
		return nil, err
	}

	// Validate definitions
	projects := make(ProjectsPool, 0)
	for _, d := range defs {
		if project, err := newProject(d, validate); err == nil {
			projects = append(projects, project)
		} else {
			return nil, fmt.Errorf(`initialization of project "%d" failed: %w`, d.ProjectID, err)
		}
	}

	return projects, nil
}

func mustGetProjects() ProjectsPool {
	projects, err := getProjects()
	if err != nil {
		panic(err)
	}
	return projects
}

func getProjects() (ProjectsPool, error) {
	poolLock.Lock()
	defer poolLock.Unlock()

	// Initialization is run only once per process
	if pool != nil {
		return pool, nil
	}

	// Init projects from the ENV
	if v, err := GetProjectsFrom(os.Getenv(`TEST_KBC_PROJECTS`)); err == nil { // nolint: forbidigo
		pool = v // initialization run only once
		return pool, nil
	} else {
		return nil, fmt.Errorf("invalid TEST_KBC_PROJECTS env: %w", err)
	}
}

// initProject - init test project handler and lock it.
func newProject(def Definition, validate *validator.Validate) (*Project, error) {
	if err := validate.Struct(def); err != nil {
		return nil, err
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
		return nil, fmt.Errorf(`cannot create locks dir: %w`, err)
	}

	// Get lock file name
	lockFile := def.Host + `-` + strconv.Itoa(def.ProjectID) + `.lock`
	lockPath := filepath.Join(locksDir, lockFile)

	return &Project{definition: def, lock: &sync.Mutex{}, fsLock: flock.New(lockPath)}, nil
}
