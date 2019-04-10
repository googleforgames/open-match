package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadMerged(t *testing.T) {
	t.Run("can merge multiple files", testCanReadAndMerge)
	t.Run("can watch changes in multiple layers", testCanWatchChanges)
}

const waitAfterConfigChange = 1 * time.Second

var commonTestData = []string{
	"first.yaml",
	"x: 123\ny: 456",

	"second.yaml",
	"x: 666",
}

// Use in pair and keep in sync with commonTestData
func commonAssertion(cfg View) error {
	for _, testcase := range []struct {
		key      string
		expected int
	}{
		// Expect original value from 1st layer
		{"y", 456},

		// Expect overriden value from 2nd layer
		{"x", 666},
	} {
		if val := cfg.GetInt(testcase.key); val != testcase.expected {
			return fmt.Errorf("%q = %d, expected %d", testcase.key, val, testcase.expected)
		}
	}
	return nil
}

func testCanReadAndMerge(t *testing.T) {
	tempdir, testfiles, err := bootstrap(commonTestData...)
	if err != nil {
		t.Fatal("cannot bootstrap test files:", err)
	}
	defer os.RemoveAll(tempdir)

	cfg, err := readMerged(testfiles...)
	if err != nil {
		t.Fatalf("cannot load config, %s", err)
	}

	if err := commonAssertion(cfg); err != nil {
		t.Fatal(err)
	}
}

func testCanWatchChanges(t *testing.T) {
	tempdir, testfiles, err := bootstrap(commonTestData...)
	if err != nil {
		t.Fatal("cannot bootstrap test files:", err)
	}
	defer os.RemoveAll(tempdir)

	cfg, err := readMerged(testfiles...)
	if err != nil {
		t.Fatalf("cannot load config, %s", err)
	}

	if err := commonAssertion(cfg); err != nil {
		t.Fatal(err)
	}

	// ------------------------------------------------
	// Modify 'x' on 2nd layer:
	// 'x' is overriden on 2nd layer and must receive modified value)
	x := 999
	secondYaml := fmt.Sprintf("x: %d", x)
	if err := ioutil.WriteFile(testfiles[1], []byte(secondYaml), 0666); err != nil {
		t.Fatalf("could not modify second layer (temp file %q): %v", testfiles[1], err)
	}
	time.Sleep(waitAfterConfigChange)

	if val := cfg.GetInt("x"); val != x {
		t.Errorf("x = %d, expected %d", val, x)
	}

	// ------------------------------------------------
	// Modify 'x' and 'y' on 1st layer:
	// 'x' must remain overriden on 2nd layer)
	y := 654
	firstYaml := fmt.Sprintf("x: 100\ny: %d", y)
	if err := ioutil.WriteFile(testfiles[0], []byte(firstYaml), 0666); err != nil {
		t.Fatalf("could not modify first layer (temp file %q): %v", testfiles[1], err)
	}
	time.Sleep(waitAfterConfigChange)

	// Expect modified value taken from 1st layer
	if val := cfg.GetInt("y"); val != y {
		t.Errorf("y = %d, expected %d", val, y)
	}

	// Expect modified overriden value taken from 2nd layer
	if val := cfg.GetInt("x"); val != x {
		t.Errorf("x = %d, expected %d", val, x)
	}

	// ------------------------------------------------
	// Modify 2nd layer several times
	x = 3
	for i := 0; i <= x; i++ {
		secondYaml := fmt.Sprintf("x: %d", i)
		if err := ioutil.WriteFile(testfiles[1], []byte(secondYaml), 0666); err != nil {
			t.Fatalf("could not modify second layer (temp file %q): %v", testfiles[1], err)
		}
	}
	time.Sleep(waitAfterConfigChange)

	if val := cfg.GetInt("x"); val != x {
		t.Errorf("x = %d, expected %d", val, x)
	}

}

func bootstrap(testdata ...string) (tempdir string, testfiles []string, err error) {
	if len(testdata)%2 != 0 {
		err = errors.New("odd number of arguments")
		return
	}

	tempdir, err = ioutil.TempDir("", "")
	if err != nil {
		return "", nil, fmt.Errorf("could not create temp dir: %v", err)
	}

	for i := 0; i <= len(testdata)/2; i += 2 {
		filename, content := testdata[i], testdata[i+1]

		f := filepath.Join(tempdir, filename)
		if ferr := ioutil.WriteFile(f, []byte(content), 0666); ferr != nil {
			err = fmt.Errorf("could not create temp file %q: %v", f, ferr)
			return
		}

		testfiles = append(testfiles, f)
	}
	return
}
