package minimatch

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/open-match/internal/app/minimatch"
	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/rs/xid"

	omTesting "github.com/GoogleCloudPlatform/open-match/internal/serving/testing"
)

func TestMinimatchStartup(t *testing.T) {
	mm, closer, err := omTesting.NewMiniMatch(minimatch.CreateServerParams())
	defer closer()
	mm.Start()
	defer mm.Stop()

	feClient, err := mm.GetFrontendClient()
	if err != nil {
		t.Fatalf("cannot obtain frontend client, %s", err)
	}
	if feClient == nil {
		t.Fatal("cannot get frontend client")
	}
	beClient, err := mm.GetBackendClient()
	if err != nil {
		t.Fatalf("cannot obtain backend client, %s", err)
	}
	if beClient == nil {
		t.Fatalf("cannot get backend client, %v", beClient)
	}

	//var lastPlayer *pb.Player
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		player := createFakePlayer()
		//lastPlayer = player
		_, err = feClient.CreatePlayer(ctx, &pb.CreatePlayerRequest{
			Player: player,
		})
		if err != nil {
			t.Errorf("request error, %s", err)
		}
	}
	// TODO: The following code would be next but there's an infinite loop that hasn't been diagnosed yet. Missing bounds checking?
	/*
		match, err := beClient.CreateMatch(ctx, &pb.MatchObject{
			Id: lastPlayer.Id,
		})
		if err != nil {
			t.Errorf("request error, %s", err)
		}
		if match.Id != "1" {
			t.Errorf("request error, %v", match)
		}
	*/
}

func createFakePlayer() *pb.Player {
	properties := make(map[string]interface{})
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
	if propBytes, err := json.Marshal(properties); err != nil {
		panic(err)
	} else {
		return &pb.Player{
			Id:         xid.New().String(),
			Properties: string(propBytes),
		}
	}
}
