package testproject_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/keboola/go-utils/pkg/testproject"
	"github.com/stretchr/testify/assert"
)

func ExampleGetTestProject() {
	// In some test ...
	t := &testing.T{}

	// There is TEST_KBC_PROJECTS environment variable.
	// Format is: <storage_api_host>|<project_id>|<token>;...
	def1 := "connection.keboola.com|1234|1234-abcdef;"
	def2 := "connection.keboola.com|3456|3456-abcdef;"
	def3 := "connection.keboola.com|5678|5678-abcdef;"
	_ = os.Setenv("TEST_KBC_PROJECTS", def1+def2+def3)

	// Acquire exclusive access to the project.
	project1 := testproject.GetTestProject(t)
	fmt.Printf("Project %d locked.\n", project1.ID())
	project2 := testproject.GetTestProject(t)
	fmt.Printf("Project %d locked.\n", project2.ID())
	project3 := testproject.GetTestProject(t)
	fmt.Printf("Project %d locked.\n", project3.ID())

	// Project lock will be automatically released at the end of the test.

	// Output:
	// Project 1234 locked.
	// Project 3456 locked.
	// Project 5678 locked.
}

func TestGetTestProject_Empty(t *testing.T) {
	_ = os.Setenv("TEST_KBC_PROJECTS", "")
	assert.PanicsWithError(t, `please specify one or more Keboola Connection testing projects by TEST_KBC_PROJECTS env, in format "<storage_api_host>|<project_id>|<token>;..."`, func() {
		t := &testing.T{}
		_ = testproject.GetTestProject(t)
	})
}
