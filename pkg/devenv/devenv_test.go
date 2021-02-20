package devenv

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

var tmpDir string

func TestMain(m *testing.M) {
	setUp()

	code := m.Run()

	tearDown()

	os.Exit(code)
}

func setUp() {
	dir, err := ioutil.TempDir("", "devenv")
	if err != nil {
		fmt.Printf("Error creating temp dir %q: %s\n", tmpDir, err)
		os.Exit(1)
	}

	tmpDir = dir

}

func tearDown() {

}

func TestStackTemplate(t *testing.T) {
	output, err := StackTemplate("foo", "bar")
	if err != nil {
		t.Errorf("Failed rendering template: %s", err)
	}

	fmt.Printf("Rendered template:\n%s", output)
}
