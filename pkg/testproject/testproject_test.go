package testproject

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockedT implements TInterface for tests, represents *testing.T.
type mockedT struct {
	cleanup []func()
}

func (v *mockedT) Cleanup(f func()) {
	v.cleanup = append(v.cleanup, f)
}

func ExampleGetTestProject() {
	// Note: For real use call the "GetTestProject" function,
	// to get a testing project from the "TEST_KBC_PROJECTS" environment variable.
	// Here, the "projects.GetTestProject" method is called to make it testable and without global variables.
	projects := MustGetProjectsFrom(projectsForTest())

	// Acquire exclusive access to the project.
	project1, unlockFn1, _ := projects.GetTestProject()
	defer unlockFn1()
	fmt.Printf("Project %d locked.\n", project1.ID())

	project2, unlockFn2, _ := projects.GetTestProject()
	defer unlockFn2()
	fmt.Printf("Project %d locked.\n", project2.ID())

	project3, unlockFn3, _ := projects.GetTestProject()
	defer unlockFn3()
	fmt.Printf("Project %d locked.\n", project3.ID())

	// The returned UnlockFn function must be called to free project, when the project is no longer used (e.g. defer unlockFn())

	// See also ExampleGetTestProjectForTest for usage in a test.

	// Output:
	// Project 1234 locked.
	// Project 3456 locked.
	// Project 5678 locked.
}

func ExampleGetTestProjectForTest() {
	t := &mockedT{}

	// Note: For real use call the "GetTestProject" function,
	// to get a testing project from the "TEST_KBC_PROJECTS" environment variable.
	// Here, the "projects.GetTestProject" method is called to make it testable and without global variables.
	projects := MustGetProjectsFrom(projectsForTest())

	// Acquire exclusive access to the project.
	project1, _ := projects.GetTestProjectForTest(t)
	fmt.Printf("Project %d locked.\n", project1.ID()) //nolint:forbidigo

	project2, _ := projects.GetTestProjectForTest(t)
	fmt.Printf("Project %d locked.\n", project2.ID()) //nolint:forbidigo

	project3, _ := projects.GetTestProjectForTest(t)
	fmt.Printf("Project %d locked.\n", project3.ID()) //nolint:forbidigo

	// Project lock will be automatically released at the end of the test.
	for _, f := range t.cleanup {
		f()
	}

	// Output:
	// Project 1234 locked.
	// Project 3456 locked.
	// Project 5678 locked.
}

func ExampleWithStagingStorage() {
	// Note: For real use call the "GetTestProject" function,
	// to get a testing project from the "TEST_KBC_PROJECTS" environment variable.
	// Here, the "projects.GetTestProject" method is called to make it testable and without global variables.
	projects := MustGetProjectsFrom(projectsForTest())

	// Acquire exclusive access to the project.
	project, unlockFn, _ := projects.GetTestProject(WithStagingStorage("abs"))
	defer unlockFn()
	fmt.Printf("Project %d locked.\n", project.ID())
	fmt.Printf("Staging storage: %s.\n", project.StagingStorage())

	// Output:
	// Project 3456 locked.
	// Staging storage: abs.
}

func ExampleGetTestProject_second() {
	// Note: For real use call the "GetTestProject" function,
	// to get a testing project from the "TEST_KBC_PROJECTS" environment variable.
	// Provide also "TEST_KBC_PROJECTS_LOCK_HOST" and "TEST_KBC_PROJECTS_LOCK_PASSWORD" variables to connect into redis.
	// Here, the "projects.GetTestProject" method is called to make it testable and without global variables.
	// os.Setenv(TestKbcProjectsLockHostKey, "redis:6379")
	// os.Setenv(TestKbcProjectsLockPasswordKey, "testing")
	// defer func() {
	//	os.Unsetenv(TestKbcProjectsLockHostKey)
	//	os.Unsetenv(TestKbcProjectsLockPasswordKey)
	// }()
	projects, err := GetProjectsFrom(projectsForTest())
	if err != nil {
		fmt.Println("redis is not up.")
		return
	}

	// Acquire exclusive access to the project.
	project1, unlockFn1, _ := projects.GetTestProject()
	defer unlockFn1()
	fmt.Printf("Project %d locked.\n", project1.ID())

	project2, unlockFn2, _ := projects.GetTestProject()
	defer unlockFn2()
	fmt.Printf("Project %d locked.\n", project2.ID())

	project3, unlockFn3, _ := projects.GetTestProject()
	defer unlockFn3()
	fmt.Printf("Project %d locked.\n", project3.ID())

	// The returned UnlockFn function must be called to free project, when the project is no longer used (e.g. defer unlockFn())

	// Output:
	// Project 1234 locked.
	// Project 3456 locked.
	// Project 5678 locked.
}

func TestGetTestProject_WithRedisLock(t *testing.T) {
	t.Parallel()
	host := os.Getenv(TestKbcProjectsLockHostKey)     // nolint: forbidigo
	password := os.Getenv(TestKbcProjectsLockHostKey) // nolint: forbidigo
	if host == "" && password == "" {
		t.Skip("no redis credentials provided")
	}

	pool, err := GetProjectsFrom(projectsForTest())
	require.NoError(t, err)
	project1, unlockFn1, _ := pool.GetTestProject(WithStagingStorageABS())
	defer unlockFn1()
	assert.Equal(t, 3456, project1.ID())
}

func TestGetTestProject_WithStagingStorage(t *testing.T) {
	t.Parallel()
	project1, unlockFn1, _ := MustGetProjectsFrom(projectsForTest()).GetTestProject(WithStagingStorageABS())
	defer unlockFn1()
	assert.Equal(t, 3456, project1.ID())
}

func TestGetTestProject_WithSnowflakeBackend(t *testing.T) {
	t.Parallel()
	project1, unlockFn1, _ := MustGetProjectsFrom(projectsForTest()).GetTestProject(WithSnowflakeBackend())
	defer unlockFn1()
	assert.Equal(t, 3456, project1.ID())
}

func TestGetTestProject_WithBigQueryBackend(t *testing.T) {
	t.Parallel()
	project1, unlockFn1, _ := MustGetProjectsFrom(projectsForTest()).GetTestProject(WithBigQueryBackend())
	defer unlockFn1()
	assert.Equal(t, 1234, project1.ID())
}

func TestGetTestProject_NoProjectForStagingStorage(t *testing.T) {
	t.Parallel()
	projects, err := GetProjectsFrom(`[{"project": 5678,"backend":"bigquery", "host": "foo.keboola.com", "token": "bar", "stagingStorage": "s3"}]`)
	assert.NoError(t, err)
	_, _, err = projects.GetTestProject(WithStagingStorage("gcs"))
	assert.ErrorContains(t, err, `no compatible test project found (staging storage gcs)`)
}

func TestGetProjectsFrom_EmptyString(t *testing.T) {
	t.Parallel()
	_, err := GetProjectsFrom("")
	assert.ErrorContains(t, err, `please specify one or more Keboola Connection testing projects in format '[{"host":"","token":"","project":"","stagingStorage":""}]'`)
}

func TestGetProjectsFrom_EmptyArray(t *testing.T) {
	t.Parallel()
	_, err := GetProjectsFrom("[]")
	assert.ErrorContains(t, err, `please specify one or more Keboola Connection testing projects in format '[{"host":"","token":"","project":"","stagingStorage":""}]'`)
}

func TestGetProjectsFrom_MissingToken(t *testing.T) {
	t.Parallel()
	_, err := GetProjectsFrom(`[{"project": 5678,"backend":"bigquery", "host": "connection.keboola.com", "stagingStorage": "s3"}]`)
	assert.ErrorContains(t, err, `initialization of project "5678" failed: Key: 'Definition.Token' Error:Field validation for 'Token' failed on the 'required' tag`)
}

func projectsForTest() string {
	projects := []Definition{
		{
			Host:           "connection.keboola.com",
			Token:          "1234-abcdef",
			Backend:        BackendBigQuery,
			StagingStorage: StagingStorageS3,
			ProjectID:      1234,
		},
		{
			Host:           "connection.north-europe.azure.keboola.com",
			Token:          "3456-abcdef",
			Backend:        BackendSnowflake,
			StagingStorage: StagingStorageABS,
			ProjectID:      3456,
		},
		{
			Host:           "connection.keboola.com",
			Token:          "5678-abcdef",
			Backend:        BackendBigQuery,
			StagingStorage: StagingStorageS3,
			ProjectID:      5678,
		},
	}
	j, err := json.Marshal(projects)
	if err != nil {
		panic(err)
	}
	return string(j)
}
