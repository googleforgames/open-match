package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/tidwall/gjson"

	"github.com/GoogleCloudPlatform/open-match/internal/pb"
)

func readProfile(filename string) (*pb.MatchObject, error) {
	jsonFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file \"%s\": %s", filename, err.Error())
	}
	defer jsonFile.Close()

	// parse json data and remove extra whitespace before sending to the backend.
	jsonData, _ := ioutil.ReadAll(jsonFile) // this reads as a byte array
	buffer := new(bytes.Buffer)             // convert byte array to buffer to send to json.Compact()
	if err := json.Compact(buffer, jsonData); err != nil {
		dirLog.WithError(err).WithField("filename", filename).Warn("error compacting profile json")
	}

	jsonProfile := buffer.String()

	profileName := "fallback-name"
	if gjson.Get(jsonProfile, "name").Exists() {
		profileName = gjson.Get(jsonProfile, "name").String()
	}

	pbProfile := &pb.MatchObject{
		Id:         profileName,
		Properties: jsonProfile,
	}
	return pbProfile, nil
}

func countPlayers(match *pb.MatchObject) int64 {
	var n int64
	for _, p := range match.Pools {
		if p.Stats != nil {
			n += p.Stats.Count
		}
	}
	return n
}

func isPartial(match *pb.MatchObject) bool {
	for _, r := range match.Rosters {
		for _, p := range r.Players {
			if p.Id == "" {
				return true
			}
		}
	}
	return false
}

func splitSlots(match *pb.MatchObject) (filled []*pb.Roster, empty []*pb.Roster) {
	for _, r := range match.Rosters {
		var f []*pb.Player
		var e []*pb.Player

		for _, p := range r.Players {
			player := *p
			if player.Id != "" {
				f = append(f, &player)
			} else {
				e = append(e, &player)
			}
		}

		if len(f) > 0 {
			filled = append(filled, &pb.Roster{Name: r.Name, Players: f})
		}
		if len(e) > 0 {
			empty = append(empty, &pb.Roster{Name: r.Name, Players: e})
		}
	}
	return
}

func collectPlayerIds(rosters []*pb.Roster) (ids []string) {
	for _, r := range rosters {
		for _, p := range r.Players {
			if p.Id != "" {
				ids = append(ids, p.Id)
			}
		}
	}
	return
}
