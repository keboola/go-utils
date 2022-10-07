package testproject

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

//nolint:paralleltest
func TestGetTestProjectForTest(t *testing.T) {
	// There is TEST_KBC_PROJECTS environment variable.
	// Format is: <storage_api_host>|<project_id>|<token>;...
	def1 := "connection.keboola.com|1234|1234-abcdef;"
	def2 := "connection.keboola.com|3456|3456-abcdef;"
	def3 := "connection.keboola.com|5678|5678-abcdef;"
	resetProjects()
	_ = os.Setenv("TEST_KBC_PROJECTS", def1+def2+def3) //nolint:forbidigo

	// Acquire exclusive access to the project.
	project1, unlockFn1, _ := GetTestProject()
	defer unlockFn1()
	fmt.Printf("Project %d locked.\n", project1.ID()) //nolint:forbidigo
	project2, unlockFn2, _ := GetTestProject()
	defer unlockFn2()
	fmt.Printf("Project %d locked.\n", project2.ID()) //nolint:forbidigo
	project3, unlockFn3, _ := GetTestProject()
	defer unlockFn3()
	fmt.Printf("Project %d locked.\n", project3.ID()) //nolint:forbidigo

	// Project lock will be automatically released at the end of the test.

	// Output:
	// Project 1234 locked.
	// Project 3456 locked.
	// Project 5678 locked.
}

func ExampleGetTestProject() {
	// There is TEST_KBC_PROJECTS environment variable.
	// Format is: <storage_api_host>|<project_id>|<token>;...
	def1 := "connection.keboola.com|1234|1234-abcdef;"
	def2 := "connection.keboola.com|3456|3456-abcdef;"
	def3 := "connection.keboola.com|5678|5678-abcdef;"
	_ = os.Setenv("TEST_KBC_PROJECTS", def1+def2+def3)

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
	assert.ErrorContains(t, err, `please specify one or more Keboola Connection testing projects by TEST_KBC_PROJECTS env, in format "<storage_api_host>|<project_id>|<token>;..."`)
}

//nolint:paralleltest
func TestGetTestProject_Empty(t *testing.T) {
	resetProjects()
	_ = os.Setenv("TEST_KBC_PROJECTS", "") //nolint:forbidigo
	_, _, err := GetTestProject()
	assert.ErrorContains(t, err, `please specify one or more Keboola Connection testing projects by TEST_KBC_PROJECTS env, in format "<storage_api_host>|<project_id>|<token>;..."`)
}
