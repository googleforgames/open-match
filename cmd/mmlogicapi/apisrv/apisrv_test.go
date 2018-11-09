package apisrv

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

var (
	a1 [5]string
	a2 [4]string
	i  [2]string
	u  [7]string
	d  [3]string
)

type stringOperation func([]string, []string) []string

func TestMain(m *testing.M) {

	a1 = [5]string{"a", "b", "c", "d", "e"}
	a2 = [4]string{"b", "c", "f", "g"}
	i = [2]string{"b", "c"}
	u = [7]string{"a", "b", "c", "d", "e", "f", "g"}
	d = [3]string{"a", "d", "e"}

	flag.Parse()
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestStringOperations(t *testing.T) {
	t.Run("Difference: valid slices", testStringOperation(difference, a1[:], a2[:], d[:]))
	t.Run("Difference: remove nil", testStringOperation(difference, a1[:], nil, a1[:]))
	t.Run("Difference: remove from nil", testStringOperation(difference, nil, a2[:], nil))
	t.Run("Union: valid slices", testStringOperation(union, a1[:], a2[:], u[:]))
	t.Run("Union: nil first", testStringOperation(union, nil, a2[:], a2[:]))
	t.Run("Union: nil second", testStringOperation(union, a1[:], nil, a1[:]))
	t.Run("Intersection: valid slices", testStringOperation(intersection, a1[:], a2[:], i[:]))
	t.Run("Intersection: nil first", testStringOperation(intersection, a1[:], nil, nil))
	t.Run("Intersection: nil second", testStringOperation(intersection, nil, a2[:], nil))
}

func testStringOperation(thisFunc stringOperation, in1 []string, in2 []string, outputExpected []string) func(*testing.T) {
	return func(t *testing.T) {
		t.Logf("func(%-15s , %-15s) = %v", fmt.Sprintf("%v", in1), fmt.Sprintf("%v", in2), outputExpected)
		outputActual := thisFunc(in1, in2)

		if len(outputActual) != len(outputExpected) {
			t.Errorf("Length of output string slice incorrect, got %v, want %v", len(outputActual), outputExpected)
		}
		for x := 0; x < len(outputActual); x++ {
			found := false
			for y := 0; y < len(outputExpected) && found != true; y++ {
				if outputActual[x] == outputExpected[y] {
					found = true
				}
			}
			if !found {
				t.Errorf("Element of output string slice incorrect, no %v in %v", outputActual[x], outputExpected)

			}

		}
	}
}
