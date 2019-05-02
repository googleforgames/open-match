// Copyright 2018 Google LLC
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

// Package player is a module used for generating stubbed players to put into the matchmaking pool.
package player

import (
	"bufio"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/xid"
)

var (
	seed      = rand.NewSource(time.Now().UnixNano())
	random    = rand.New(seed)
	percents  = []float64{}
	cities    = []string{}
	pingStats = map[string]map[string]map[string]float64{}
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Choose a city based on stacked percentages in city.percent
func pick() string {
	p := percents[0]
	x := random.Float64()
	i := 0
	for x > p {
		i = i + 1
		// If you're percents are not stacked
		//p = p + percents[i]
		p = percents[i]
	}
	return cities[i]
}

// Generates a random integer in a normal distribution
func normalDist(avg float64, min float64, max float64, stdev float64) int {
	sample := (rand.NormFloat64() * stdev) + avg
	switch {
	case sample > max:
		sample = max
	case sample < min:
		sample = min
	}

	return int(sample)
}

// Pings come in a string format of '%d.%dms'
// Remove last two characters ('ms') of the string and convert the rest to float64
func pingToFloat(s string) float64 {
	r, err := strconv.ParseFloat(s[:len(s)-2], 64)
	check(err)
	return r
}

// New initializes a new player generator
func New() {
	pingFiles, err := filepath.Glob("*.ping")
	if err != nil {
		log.Fatal(err)
	}
	percentFile, err := os.Open("city.percent")
	if err != nil {
		log.Fatal(err)
	}
	defer percentFile.Close()
	scanner := bufio.NewScanner(percentFile)

	// Read in the percentages file
	for scanner.Scan() {
		//fmt.Println(reflect.TypeOf(scanner.Text()))
		//fmt.Println(scanner.Text())
		words := strings.Fields(scanner.Text())
		percent, _ := strconv.ParseFloat(words[0], 64)
		percents = append(percents, percent)
		city := strings.Join(words[1:], " ")
		cities = append(cities, city)
	}
	//fmt.Println(percents, cities)

	// Read in ping files
	for _, pingFilename := range pingFiles {
		region := strings.Split(pingFilename, ".")[0]
		pingFile, err := os.Open(pingFilename)
		check(err)
		defer pingFile.Close()

		// Init map for this region
		pingStats[region] = map[string]map[string]float64{}

		scanner = bufio.NewScanner(pingFile)
		for scanner.Scan() {
			words := strings.Split(scanner.Text(), "\t")
			wl := len(words)
			city := words[0]
			pingStats[region][city] = map[string]float64{}
			cur := pingStats[region][city]
			cur["avg"] = pingToFloat(words[wl-6])
			cur["min"] = pingToFloat(words[wl-4])
			cur["max"] = pingToFloat(words[wl-3])
			cur["std"] = pingToFloat(words[wl-2])
		}
	}
}

// Generate a player
// For PoC, we're flattening the JSON so it can be easily indexed in Redis.
// Flattened keys are joined using periods.
// That should be abstracted out of this level and into the db storage module
func Generate() (string, map[string]interface{}) {

	id := xid.New().String()
	city := pick()
	properties := map[string]interface{}{}

	// Generate some fake latencies
	regions := map[string]int{}
	for region := range pingStats {
		//fmt.Print(region, " ")
		regions[region] = normalDist(
			pingStats[region][city]["avg"],
			pingStats[region][city]["min"],
			pingStats[region][city]["max"],
			pingStats[region][city]["std"],
		)
	}
	properties["region"] = regions

	// Insert other properties here
	// For example, a random skill modeled on a normal distribution
	properties["mmr"] = map[string]int{"rating": normalDist(1500, -1000, 4000, 350)}

	// For properties that are just flags, the key is the important bit.
	// It's existance denotes a boolean true value.
	// Just use an epoch timestamp as the value.
	now := int(time.Now().Unix())
	properties["char"] = map[string]int{
		"paladin":   now,
		"knight":    now,
		"barbarian": now,
	}
	properties["map"] = map[string]int{
		"oasis": now,
		"dirt":  now,
	}
	properties["mode"] = map[string]int{
		"ctf":          now,
		"battleroyale": now,
	}

	return id, properties
}
