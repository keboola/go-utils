package testproject

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func initProjectsDefinition() string {
	def1 := &Project{
		Host:      "connection.keboola.com",
		Token:     "1234-abcdef",
		Provider:  "aws",
		ProjectID: 1234,
	}
	def2 := &Project{
		Host:      "connection.north-europe.azure.keboola.com",
		Token:     "3456-abcdef",
		Provider:  "azure",
		ProjectID: 3456,
	}
	def3 := &Project{
		Host:      "connection.keboola.com",
		Token:     "5678-abcdef",
		Provider:  "aws",
		ProjectID: 5678,
	}
	j, _ := json.Marshal([]*Project{def1, def2, def3})
	return string(j)
}

//nolint:paralleltest
func TestGetTestProjectForTest(t *testing.T) {
	// There is json-encoded TEST_KBC_PROJECTS environment variable.
	_ = os.Setenv("TEST_KBC_PROJECTS", initProjectsDefinition()) //nolint:forbidigo

	// Acquire exclusive access to the project.
	project1, unlockFn1, err := GetTestProject()
	assert.NoError(t, err)
	defer unlockFn1()
	fmt.Printf("Project %d locked.\n", project1.ID()) //nolint:forbidigo
	project2, unlockFn2, err := GetTestProject()
	assert.NoError(t, err)
	defer unlockFn2()
	fmt.Printf("Project %d locked.\n", project2.ID()) //nolint:forbidigo
	project3, unlockFn3, err := GetTestProject()
	assert.NoError(t, err)
	defer unlockFn3()
	fmt.Printf("Project %d locked.\n", project3.ID()) //nolint:forbidigo

	// Project lock will be automatically released at the end of the test.

	// Output:
	// Project 1234 locked.
	// Project 3456 locked.
	// Project 5678 locked.
}

//nolint:paralleltest
func TestGetTestProjectForTestProvider(t *testing.T) {
	_ = os.Setenv("TEST_KBC_PROJECTS", initProjectsDefinition()) //nolint:forbidigo

	// Acquire exclusive access to the project.
	project1, unlockFn1, _ := GetTestProject(WithProvider("azure"))
	defer unlockFn1()
	assert.Equal(t, 3456, project1.ProjectID)
}

func ExampleGetTestProject() {
	_ = os.Setenv("TEST_KBC_PROJECTS", initProjectsDefinition()) //nolint:forbidigo

	// Acquire exclusive access to the project.
	project1, unlockFn1, _ := GetTestProject()
	defer unlockFn1()
	fmt.Printf("Project %d locked.\n", project1.ID())
	project2, unlockFn2, _ := GetTestProject()
	defer unlockFn2()
	fmt.Printf("Project %d locked.\n", project2.ID())
	project3, unlockFn3, _ := GetTestProject()
	defer unlockFn3()
	fmt.Printf("Project %d locked.\n", project3.ID())

	// The returned UnlockFn function must be called to free project, when the project is no longer used (e.g. defer unlockFn())

	// See also TestGetTestProjectForTest for usage in a test.

	// Output:
	// Project 1234 locked.
	// Project 3456 locked.
	// Project 5678 locked.
}

//nolint:paralleltest
func TestGetTestProjectForTest_Empty(t *testing.T) {
	resetProjects()
	_ = os.Setenv("TEST_KBC_PROJECTS", "") //nolint:forbidigo
	_, err := GetTestProjectForTest(t)
	assert.ErrorContains(t, err, `please specify one or more Keboola Connection testing projects by TEST_KBC_PROJECTS env, in format '[{"host":"","token":"","project":"","provider":""}]'`)
}

//nolint:paralleltest
func TestGetTestProject_Empty(t *testing.T) {
	resetProjects()
	_ = os.Setenv("TEST_KBC_PROJECTS", "[]") //nolint:forbidigo
	_, _, err := GetTestProject()
	assert.ErrorContains(t, err, `please specify one or more Keboola Connection testing projects by TEST_KBC_PROJECTS env, in format '[{"host":"","token":"","project":"","provider":""}]'`)
}

//nolint:paralleltest
func TestGetTestProjectProvider_Missing(t *testing.T) {
	resetProjects()
	_ = os.Setenv("TEST_KBC_PROJECTS", initProjectsDefinition()) //nolint:forbidigo

	_, _, err := GetTestProject(WithProvider("gcp"))
	assert.ErrorContains(t, err, `no test project for provider gcp`)
}
