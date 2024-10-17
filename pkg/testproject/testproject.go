// Package testproject implements locking of Keboola Projects for E2E parallel tests.
//
// Project is locked:
// - at locker level (redislock) OR
// - at the host level (flock.Flock) AND at the goroutines level (sync.Mutex)
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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslation "github.com/go-playground/validator/v10/translations/en"
)

const (
	StagingStorageABS = "abs"
	StagingStorageGCS = "gcs"
	StagingStorageS3  = "s3"

	BackendSnowflake               = "snowflake"
	BackendBigQuery                = "bigquery"
	TestKbcProjectsFileKey         = "TEST_KBC_PROJECTS_FILE"
	TestKbcProjectsLockDirNameKey  = "TEST_KBC_PROJECTS_LOCK_DIR_NAME"
	TestKbcProjectsLockHostKey     = "TEST_KBC_PROJECTS_LOCK_HOST"
	TestKbcProjectsLockPasswordKey = "TEST_KBC_PROJECTS_LOCK_PASSWORD"
	TestKbcProjectsLockTLSKey      = "TEST_KBC_PROJECTS_LOCK_TLS"
)

const QueueV1 = "v1"

var pool *ProjectsPool       // nolint gochecknoglobals
var poolLock = &sync.Mutex{} // nolint gochecknoglobals

type locker interface {
	newForProject(p *Project) projectLocker
}

type projectLocker interface {
	tryLock() bool
	unlock()
	isLocked() bool
}

// ProjectsPool a group of testing projects.
type ProjectsPool []*Project

// Project represents a testing project for E2E tests.
type Project struct {
	definition Definition
	locker     projectLocker
}

// Definition is project Definition parsed from the ENV.
type Definition struct {
	Host                 string `json:"host" validate:"required"`
	Token                string `json:"token" validate:"required"`
	StagingStorage       string `json:"stagingStorage" validate:"required"`
	Backend              string `json:"backend" validate:"required"`
	ProjectID            int    `json:"project" validate:"required"`
	LegacyTransformation bool   `json:"legacyTransformation"`
	Queue                string `json:"queue,omitempty"`
}

// UnlockFn must be called if the project is no longer used.
type UnlockFn func()

// Option for the GetTestProjectForTest and GetTestProject functions.
type Option func(c *config)

// config for the GetTestProjectForTest and GetTestProject functions.
type config struct {
	backend              string
	stagingStorage       string
	legacyTransformation bool
	queueV1              bool
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

func WithQueueV1() Option {
	return func(c *config) {
		c.queueV1 = true
	}
}

func WithSnowflakeBackend() Option {
	return func(c *config) {
		c.backend = BackendSnowflake
	}
}

func WithBigQueryBackend() Option {
	return func(c *config) {
		c.backend = BackendBigQuery
	}
}

func WithLegacyTransformation() Option {
	return func(c *config) {
		c.legacyTransformation = true
	}
}

func (c *config) IsCompatible(p *Project) bool {
	matchStagingStorage := len(c.stagingStorage) == 0 || p.definition.StagingStorage == c.stagingStorage

	matchQueue := (p.definition.Queue == QueueV1) == c.queueV1 // QueueV2 is required, if QueueV1 is not explicitly requested

	matchBackend := len(c.backend) == 0 || p.definition.Backend == c.backend

	matchLegacyTransformation := !c.legacyTransformation || p.definition.LegacyTransformation == c.legacyTransformation

	return matchStagingStorage && matchQueue && matchBackend && matchLegacyTransformation
}

func (c *config) String() string {
	out := []string{}
	if len(c.stagingStorage) > 0 {
		out = append(out, fmt.Sprintf("staging storage %s", c.stagingStorage))
	}

	if c.queueV1 {
		out = append(out, "queue v1")
	}

	if len(c.backend) > 0 {
		out = append(out, fmt.Sprintf("backend %s", c.backend))
	}

	if c.legacyTransformation {
		out = append(out, fmt.Sprintf("legacy transformation %v", c.legacyTransformation))
	}

	return "(" + strings.Join(out, ", ") + ")"
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

func GetTestProjectInPath(path string, opts ...Option) (*Project, UnlockFn, error) {
	return mustGetProjectsInPath(path).GetTestProject(opts...)
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
			if c.IsCompatible(p) {
				if p.locker.tryLock() {
					unlockFn := func() {
						p.locker.unlock()
					}
					return p, unlockFn, nil
				}

				anyProjectFound = true
			}
		}

		if !anyProjectFound {
			return nil, nil, fmt.Errorf(`no compatible test project found %s`, c.String())
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

// Backend returns backend of the project Definition.
func (p *Project) Backend() string {
	p.assertLocked()
	return p.definition.Backend
}

// LegacyTransformation returns support of legacy transformations of the project Definition.
func (p *Project) LegacyTransformation() bool {
	p.assertLocked()
	return p.definition.LegacyTransformation
}

func (p *Project) assertLocked() {
	if !p.locker.isLocked() {
		panic(fmt.Errorf(`test project "%d" is not locked`, p.definition.ProjectID))
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

	locker, err := newLocker()
	if err != nil {
		return nil, err
	}

	// Validate definitions
	pool := make(ProjectsPool, 0)
	for _, d := range defs {
		if project, err := newProject(locker, d, validate); err == nil {
			pool = append(pool, project)
		} else {
			return pool, fmt.Errorf(`initialization of project "%d" failed: %w`, d.ProjectID, err)
		}
	}

	return pool, nil
}

func mustGetProjects() *ProjectsPool {
	projects, err := getProjects("")
	if err != nil {
		panic(err)
	}
	return projects
}

func mustGetProjectsInPath(path string) *ProjectsPool {
	projects, err := getProjects(path)
	if err != nil {
		panic(err)
	}
	return projects
}

// getProjects loads projects from provided file by path or environment variable TEST_KBC_PROJECTS_FILE.
func getProjects(path string) (*ProjectsPool, error) {
	poolLock.Lock()
	defer poolLock.Unlock()

	// Initialization is run only once per process
	if pool != nil {
		return pool, nil
	}

	projectsFile := path
	if projectsFile == "" {
		projectsFile = os.Getenv(TestKbcProjectsFileKey) // nolint: forbidigo
		if projectsFile == "" {
			return nil, fmt.Errorf("please set TEST_KBC_PROJECTS_FILE environment variable")
		}
	}

	if !filepath.IsAbs(projectsFile) {
		return nil, fmt.Errorf("the path to projects.json file should be absolute, not relative, got %s", projectsFile)
	}

	// Init projects from the json projects file
	projects, err := os.ReadFile(projectsFile) // nolint: forbidigo
	if err != nil {
		return nil, fmt.Errorf("error occurred during project pool setup: %w", err)
	}

	if v, err := GetProjectsFrom(string(projects)); err == nil {
		pool = &v // initialization run only once
		return pool, nil
	} else {
		return nil, fmt.Errorf("error occurred during project pool setup: %w", err)
	}
}

func newLocker() (locker, error) {
	redisHost := os.Getenv(TestKbcProjectsLockHostKey)         // nolint: forbidigo
	redisPassword := os.Getenv(TestKbcProjectsLockPasswordKey) // nolint: forbidigo
	if redisHost == "" && redisPassword == "" {
		locker, err := newFsLocker()
		return locker, err
	} else if redisPassword == "" {
		return nil, errors.New("redis password is required")
	}

	locker, err := newRedisLocker(redisHost, redisPassword)
	if err != nil {
		return nil, err
	}

	return locker, nil
}

// initProject - init test project handler and lock it.
func newProject(l locker, def Definition, validate *validator.Validate) (*Project, error) {
	if err := validate.Struct(def); err != nil {
		return nil, err
	}

	p := &Project{definition: def}
	projectLocker := l.newForProject(p)
	p.locker = projectLocker
	return p, nil
}
