package testproject

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTestProjectForTest(t *testing.T) {
	// There is TEST_KBC_PROJECTS environment variable.
	// Format is: <storage_api_host>|<project_id>|<token>;...
	def1 := "connection.keboola.com|1234|1234-abcdef;"
	def2 := "connection.keboola.com|3456|3456-abcdef;"
	def3 := "connection.keboola.com|5678|5678-abcdef;"
	resetProjects()
	_ = os.Setenv("TEST_KBC_PROJECTS", def1+def2+def3)

	// Acquire exclusive access to the project.
	project1 := GetTestProjectForTest(t)
	fmt.Printf("Project %d locked.\n", project1.ID())
	project2 := GetTestProjectForTest(t)
	fmt.Printf("Project %d locked.\n", project2.ID())
	project3 := GetTestProjectForTest(t)
	fmt.Printf("Project %d locked.\n", project3.ID())

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
	project1, unlockFn1 := GetTestProject()
	defer unlockFn1()
	fmt.Printf("Project %d locked.\n", project1.ID())
	project2, unlockFn2 := GetTestProject()
	defer unlockFn2()
	fmt.Printf("Project %d locked.\n", project2.ID())
	project3, unlockFn3 := GetTestProject()
	defer unlockFn3()
	fmt.Printf("Project %d locked.\n", project3.ID())

	// The returned UnlockFn function must be called to free project, when the project is no longer used (e.g. defer unlockFn())

	// See also TestGetTestProjectForTest for usage in a test.

	// Output:
	// Project 1234 locked.
	// Project 3456 locked.
	// Project 5678 locked.
}

func TestGetTestProjectForTest_Empty(t *testing.T) {
	resetProjects()
	_ = os.Setenv("TEST_KBC_PROJECTS", "")
	assert.PanicsWithError(t, `please specify one or more Keboola Connection testing projects by TEST_KBC_PROJECTS env, in format "<storage_api_host>|<project_id>|<token>;..."`, func() {
		t := &testing.T{}
		_ = GetTestProjectForTest(t)
	})
}

func TestGetTestProject_Empty(t *testing.T) {
	resetProjects()
	_ = os.Setenv("TEST_KBC_PROJECTS", "")
	assert.PanicsWithError(t, `please specify one or more Keboola Connection testing projects by TEST_KBC_PROJECTS env, in format "<storage_api_host>|<project_id>|<token>;..."`, func() {
		_, _ = GetTestProject()
	})
}
