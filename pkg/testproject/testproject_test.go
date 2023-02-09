package testproject

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestGetTestProject_WithStagingStorage(t *testing.T) {
	t.Parallel()
	project1, unlockFn1, _ := MustGetProjectsFrom(projectsForTest()).GetTestProject(WithStagingStorageABS())
	defer unlockFn1()
	assert.Equal(t, 3456, project1.ID())
}

func TestGetTestProject_WithQueueV1(t *testing.T) {
	t.Parallel()
	project1, unlockFn1, _ := MustGetProjectsFrom(projectsForTest()).GetTestProject(WithQueueV1())
	defer unlockFn1()
	assert.Equal(t, 7890, project1.ID())
}

func TestGetTestProject_NoProjectForStagingStorage(t *testing.T) {
	t.Parallel()
	projects, err := GetProjectsFrom(`[{"project": 5678, "host": "foo.keboola.com", "token": "bar", "stagingStorage": "s3"}]`)
	assert.NoError(t, err)
	_, _, err = projects.GetTestProject(WithStagingStorage("gcs"))
	assert.ErrorContains(t, err, `no compatible test project found (staging storage gcs)`)
}

func TestGetTestProject_NoProjectWithQueueV1(t *testing.T) {
	t.Parallel()
	projects, err := GetProjectsFrom(`[{"project": 5678, "host": "foo.keboola.com", "token": "bar", "stagingStorage": "s3"}]`)
	assert.NoError(t, err)
	_, _, err = projects.GetTestProject(WithQueueV1())
	assert.ErrorContains(t, err, `no compatible test project found (queue v1)`)
}

func TestGetTestProject_NoProjectWithoutQueueV1(t *testing.T) {
	t.Parallel()
	projects, err := GetProjectsFrom(`[{"project": 5678, "host": "foo.keboola.com", "token": "bar", "stagingStorage": "s3", "queue": "v1"}]`)
	assert.NoError(t, err)
	_, _, err = projects.GetTestProject()
	assert.ErrorContains(t, err, `no compatible test project found`)
}

func TestGetTestProject_NoProjectWithStagingStorageABSAndQueueV1(t *testing.T) {
	t.Parallel()
	projects, err := GetProjectsFrom(`[{"project": 5678, "host": "foo.keboola.com", "token": "bar", "stagingStorage": "s3"}]`)
	assert.NoError(t, err)
	_, _, err = projects.GetTestProject(WithStagingStorageABS(), WithQueueV1())
	assert.ErrorContains(t, err, `no compatible test project found (staging storage abs, queue v1)`)
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
	_, err := GetProjectsFrom(`[{"project": 5678, "host": "connection.keboola.com", "stagingStorage": "s3"}]`)
	assert.ErrorContains(t, err, `initialization of project "5678" failed: Key: 'Definition.Token' Error:Field validation for 'Token' failed on the 'required' tag`)
}

func projectsForTest() string {
	projects := []Definition{
		{
			Host:           "connection.keboola.com",
			Token:          "1234-abcdef",
			StagingStorage: "s3",
			ProjectID:      1234,
		},
		{
			Host:           "connection.north-europe.azure.keboola.com",
			Token:          "3456-abcdef",
			StagingStorage: "abs",
			ProjectID:      3456,
		},
		{
			Host:           "connection.keboola.com",
			Token:          "5678-abcdef",
			StagingStorage: "s3",
			ProjectID:      5678,
		},
		{
			Host:           "connection.keboola.com",
			Token:          "7890-abcdef",
			StagingStorage: "s3",
			ProjectID:      7890,
			Queue:          "v1",
		},
	}
	j, err := json.Marshal(projects)
	if err != nil {
		panic(err)
	}
	return string(j)
}
