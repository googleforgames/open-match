package mo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/gobs/pretty"
	"github.com/gogo/protobuf/jsonpb"
)

var (
	teamSize int
	numTeams int
	debug    bool
	regions  []string
	tiers    []string
)

type poolArray []*pb.PlayerPool
type ftrArray []*pb.Filter
type rosterArray []*pb.Roster

/*
//example of how to use this module
func main() {
	moChan := make(chan *pb.MatchObject)
	doneChan := make(chan bool)
	go GenerateMatchObjects(moChan, doneChan)
	for {
		select {
		case mo := <-moChan:
			pretty.PrettyPrint(mo)
		case <-doneChan:
			log.Println("all Done")
			return
		}
	}
}
*/

// GenerateMatchObjects
func GenerateMatchObjects(moChan chan *pb.MatchObject) {
	//func GenerateMatchObjects(moChan chan *pb.MatchObject, doneChan chan bool) {
	defer close(moChan)

	debug = false
	teamSize = 8
	numTeams = 2
	this := jsonpb.Marshaler{Indent: "  "}
	_ = this
	regions := []string{
		"europe-west1",
		"europe-west2",
		"europe-west3",
		"europe-west4",
	}
	tiers = []string{
		"bronze",
		"silver",
		"gold",
		"platinum",
		"diamond",
		"master",
		"grandmaster",
	}

	// Read filters from json file
	jsonfile, err := os.Open("filters.json")
	if err != nil {
		log.Fatal(err)
	}
	byteValue, err := ioutil.ReadAll(jsonfile)
	if err != nil {
		log.Fatal(err)
	}
	var pf pb.PlayerPool
	json.Unmarshal(byteValue, &pf)

	// print
	if debug {
		pretty.PrettyPrint(pf)
	}

	// Load filters into map for easy access by name
	filters := make(map[string]*pb.Filter)
	for _, filter := range pf.Filters {
		filters[filter.Name] = filter
	}

	// Make backend match object
	for _, region := range regions {
		for _, tier := range tiers {
			name := region + tier

			// Make pool on the fly from this region + tier
			pools := append(poolArray{}, &pb.PlayerPool{Name: name,
				Filters: append(ftrArray{},
					filters[region],
					filters[tier],
				)})

			// Make numTeams teams of teamSize
			rosters := rosterArray{}
			for i := 0; i < numTeams; i++ {
				rosters = append(rosters, AddPlayerSlots(&pb.Roster{Name: strconv.Itoa(i)}, *pools[0], teamSize))
			}

			mo := &pb.MatchObject{
				Id:         name,
				Properties: fmt.Sprintf("{\"testprofile\": true, \"region\": %v, \"tier\": %v}", region, tier),
				Rosters:    rosters,
				Pools:      pools,
			}
			if debug {
				// print
				pretty.PrettyPrint(mo)
			}
			// return match object
			moChan <- mo
		}
	}

	//doneChan <- true
	return
}

// GenerateMatchObjects
func ProcedurallyGenerateMatchObjects(concurrency int, moChan chan *pb.MatchObject) {
	//func GenerateMatchObjects(moChan chan *pb.MatchObject, doneChan chan bool) {
	defer close(moChan)

	debug = false
	teamSize = 8
	numTeams = 2
	minMMR := 0
	maxMMR := 4350
	filterSize := (maxMMR - minMMR) / concurrency
	minPing := 0
	maxPing := 250
	pingSize := (maxPing - minPing) / concurrency

	this := jsonpb.Marshaler{Indent: "  "}
	_ = this
	regions := []string{
		"europe-west1",
		//		"europe-west2",
		//		"europe-west3",
		//		"europe-west4",
	}

	// Read filters from json file
	jsonfile, err := os.Open("filters.json")
	if err != nil {
		log.Fatal(err)
	}
	byteValue, err := ioutil.ReadAll(jsonfile)
	if err != nil {
		log.Fatal(err)
	}
	var pf pb.PlayerPool
	json.Unmarshal(byteValue, &pf)

	// print
	if debug {
		pretty.PrettyPrint(pf)
	}

	// Load filters into map for easy access by name
	filters := make(map[string]*pb.Filter)
	for _, filter := range pf.Filters {
		filters[filter.Name] = filter
	}

	// Make backend match object
	for _, region := range regions {
		for j := 0; j < concurrency; j++ {
			for i := 0; i < concurrency; i++ {

				thisMinPing := minPing + (pingSize * j) + 1
				thisMaxPing := minPing + (pingSize * (j + 1))
				pingString := fmt.Sprintf("%v-%v", thisMinPing, thisMaxPing)
				thisMinMMR := minMMR + (filterSize * i) + 1
				thisMaxMMR := minMMR + (filterSize * (i + 1))
				rangeString := fmt.Sprintf("%v-%v", thisMinMMR, thisMaxMMR)
				name := region + "=" + pingString + "=" + rangeString

				// Make pool on the fly from this region + tier
				pools := append(poolArray{}, &pb.PlayerPool{Name: name,
					Filters: append(ftrArray{},
						&pb.Filter{Name: region + pingString,
							Attribute: "region." + region,
							Minv:      int64(thisMinPing),
							Maxv:      int64(thisMaxPing)},
						&pb.Filter{Name: rangeString,
							Attribute: "mmr.rating",
							Minv:      int64(thisMinMMR),
							Maxv:      int64(thisMaxMMR)},
					)})

				// Make numTeams teams of teamSize
				rosters := rosterArray{}
				for i := 0; i < numTeams; i++ {
					rosters = append(rosters, AddPlayerSlots(&pb.Roster{Name: strconv.Itoa(i)}, *pools[0], teamSize))
				}

				mo := &pb.MatchObject{
					Id:         name,
					Properties: fmt.Sprintf("{\"testprofile\": true, \"region\": %v, \"mmr\": %v}", region, rangeString),
					Rosters:    rosters,
					Pools:      pools,
				}
				if debug {
					// print
					pretty.PrettyPrint(mo)
				}
				// return match object
				moChan <- mo
			}
		}
	}

	//doneChan <- true
	return
}

// AddPlayerSlots add the specified number of player slots in the specified pool
func AddPlayerSlots(roster *pb.Roster, pool pb.PlayerPool, numPlayers int) *pb.Roster {
	for i := 0; i < numPlayers; i++ {
		roster.Players = append(roster.Players, &pb.Player{Pool: pool.Name})
	}
	return roster
}
