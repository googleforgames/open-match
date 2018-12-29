package main

import (
	"context"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/GoogleCloudPlatform/open-match/examples/director/pkg/backendapi"
	"github.com/GoogleCloudPlatform/open-match/internal/pb"
)

const waitOnInsufficientPlayers time.Duration = 5 * time.Second
const waitOnPartialMatch time.Duration = 5 * time.Second

func complementMatch(ctx context.Context, orig *pb.MatchObject, sink chan<- *pb.MatchObject, l *log.Entry) error {
	defer close(sink)

	if !isPartial(orig) {
		return nil
	}

	cmLog := l.WithField("origID", orig.Id)

	for mo := orig; ; {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req := buildComplementaryProfile(mo)

		cmLog.WithField("prof", *req).Debug("Calling CreateMatch() with complementary profile")
		resp, err := backendapi.CreateMatch(ctx, req)

		if err != nil && !isNotEnoughPlayers(err) {
			// Fatal error
			return err
		}

		if err != nil {
			// That's ok, let's just wait and re-try the same profile
			time.Sleep(waitOnInsufficientPlayers)
			continue
		}

		_, empty := splitSlots(resp)

		sink <- resp

		if len(empty) == 0 {
			// We're done complementing the partial match
			return nil
		}

		time.Sleep(waitOnPartialMatch)
		mo = resp
		continue
	}
}

func isNotEnoughPlayers(err error) bool {
	return strings.Contains(err.Error(), "insufficient players")
}

func buildComplementaryProfile(match *pb.MatchObject) *pb.MatchObject {
	_, emptyRosters := splitSlots(match)

	// 0. As long as MMF can create partial matches, we can keep using it
	props := match.Properties
	// 1. Figure out original profile ID
	profID := gjson.Get(props, "id").String()
	// 2. Use original filters & pools. Reduce a set of rosters to only empty slots.
	props, _ = sjson.Set(props, "properties.rosters", emptyRosters)
	// 3. Likely we want to track the history of all "addition" attempts for the backfill.
	props, _ = sjson.Set(props, "properties.origID", match.Id)

	return &pb.MatchObject{Id: profID, Properties: props}
}
