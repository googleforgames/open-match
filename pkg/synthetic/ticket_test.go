// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package synthetic

import (
	"math/rand"
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"open-match.dev/open-match/pkg/structs"
)

func TestTickets(t *testing.T) {
	rand.Seed(0)

	// Mostly tested by individual field tests, do a little sanity checking but
	// avoid just making this a change detector test.
	ticket := Tickets(1)[0]
	if _, ok := ticket.Properties.Fields["mmr.rating"]; !ok {
		t.Errorf("Ticket missing mmr.rating")
	}
	if _, ok := ticket.Properties.Fields["mode.demo"]; !ok {
		t.Errorf("Ticket missing mode.demo")
	}
	if _, ok := ticket.Properties.Fields["region.europe-east1"]; !ok {
		t.Errorf("Ticket missing region.europe-east1")
	}
}

func TestNormal(t *testing.T) {
	rand.Seed(0)

	within1stdiv := 0
	add := func(field string, value *structpb.Value) {
		if field != "myfield" {
			t.Errorf("Expected field myfield, but was %s", field)
		}
		v := value.GetNumberValue()
		if 100 < v && v < 200 {
			within1stdiv++
		}
		if v < 99 || v > 201 {
			t.Errorf("Value %v outside of clamped limits", v)
		}
	}

	f := Normal("myfield", 150, 50, 99, 201)
	for i := 0; i < 1000; i++ {
		f(add)
	}

	if within1stdiv < 640 || 720 < within1stdiv {
		t.Errorf("Expected that approx 68 percent of values would be within 1 standard deviation, got %d / 1000", within1stdiv)
	}
}

func TestFieldChoice(t *testing.T) {
	rand.Seed(0)

	choices := make(map[string]int)
	callCount := 0

	add := func(field string, value *structpb.Value) {
		v := value.GetNumberValue()
		if v != 1 {
			t.Errorf("Expected value to be 1, but got %v", v)
		}

		choices[field]++
		callCount++
	}

	f := FieldChoice(map[string]int64{
		"Foo":   600,
		"Bar":   300,
		"Baz":   100,
		"Bungo": 0,
	})

	for i := 0; i < 1000; i++ {
		callCount = 0
		f(add)

		if callCount != 1 {
			t.Errorf("Expected callCount to be 1, but got %v", callCount)
		}
	}

	if choices["Foo"] < 550 || choices["Foo"] > 650 {
		t.Errorf("choices[Foo] should be near 600, but was %d", choices["Foo"])
	}
	if choices["Bar"] < 250 || choices["Bar"] > 350 {
		t.Errorf("choices[Bar] should be near 300, but was %d", choices["Bar"])
	}
	if choices["Baz"] < 50 || choices["Baz"] > 150 {
		t.Errorf("choices[Baz] should be near 100, but was %d", choices["Baz"])
	}
	if choices["Bungo"] != 0 {
		t.Errorf("choices[Bungo] should be 0, but was %d", choices["Bungo"])
	}
}

func TestValueChoice(t *testing.T) {
	rand.Seed(0)

	choices := make(map[float64]int)
	callCount := 0

	add := func(field string, value *structpb.Value) {
		if field != "myfield" {
			t.Errorf("Expected field myfield, but was %s", field)
		}

		v := value.GetNumberValue()
		choices[v]++
		callCount++
	}

	f := ValueChoice("myfield", []*structpb.Value{
		structs.Number(0), structs.Number(1), structs.Number(2),
	}, []int64{
		0, 100, 200,
	})

	for i := 0; i < 300; i++ {
		callCount = 0
		f(add)

		if callCount != 1 {
			t.Errorf("Expected callCount to be 1, but got %v", callCount)
		}
	}

	if choices[0] != 0 {
		t.Errorf("choices[0] should be 0, but was %d", choices[0])
	}
	if choices[1] < 50 || choices[1] > 150 {
		t.Errorf("choices[1] should be near 100, but was %d", choices[1])
	}
	if choices[2] < 150 || choices[1] > 250 {
		t.Errorf("choices[2] should be near 200, but was %d", choices[2])
	}
}

func TestWeightedChoice(t *testing.T) {
	rand.Seed(0)

	choices := make([]int, 5)
	f := weightedChoice([]int64{1000, 1000, 4000, 3000, 1000})
	for i := 0; i < 10000; i++ {
		choices[f()]++
	}

	if choices[0] < 800 || choices[0] > 1200 {
		t.Errorf("choices[0] should be near 1000, but was %d", choices[0])
	}
	if choices[1] < 800 || choices[1] > 1200 {
		t.Errorf("choices[1] should be near 1000, but was %d", choices[1])
	}
	if choices[2] < 3800 || choices[2] > 4200 {
		t.Errorf("choices[2] should be near 4000, but was %d", choices[2])
	}
	if choices[3] < 2800 || choices[3] > 3200 {
		t.Errorf("choices[3] should be near 3000, but was %d", choices[3])
	}
	if choices[4] < 800 || choices[4] > 1200 {
		t.Errorf("choices[4] should be near 100, but was %d", choices[4])
	}
}

func TestWeightedChoiceLargeArray(t *testing.T) {
	rand.Seed(0)

	for i := 0; i < 1000; i++ {
		size := rand.Intn(1000) + 1
		weights := make([]int64, size)
		expected := rand.Intn(size)
		weights[expected] = 1

		actual := weightedChoice(weights)()
		if actual != expected {
			t.Errorf("Expected %d, but got %d", expected, actual)
		}
	}
}

func TestBadArtificalLatency(t *testing.T) {
	rand.Seed(0)

	dist := make([]int, 30)

	add := func(field string, value *structpb.Value) {
		v := value.GetNumberValue()

		i := int(v / 10)
		if i < len(dist) {
			dist[i]++
		}
	}

	f := BadArtificalLatency("foo", "bar", "baz")
	for i := 0; i < 1000; i++ {
		f(add)
	}

	// 40ms to 120ms
	for i := 4; i < 12; i++ {
		if dist[i] < 100 {
			t.Error("Expected that medium latencies would have lots of players.")
		}
	}
}
