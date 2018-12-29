package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	sdk "agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/signals"
	gosdk "agones.dev/agones/sdks/go"
)

const maxBufferSize = 1024

var state struct {
	gsName         string
	matchID        string
	playersRosters map[string]string
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		stop := signals.NewStopChannel()
		<-stop
		log.Println("Exit signal received, stopping everything")
		cancel()
	}()

	log.Print("Creating SDK instance")
	s, err := gosdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	defer func() {
		log.Println("Shutting down")
		err = s.Shutdown()
		if err != nil {
			log.Printf("Error doing sdk.Shutdown(): %s", err)
		}
	}()

	log.Print("Starting Health Ping")
	go doHealth(s, ctx.Done())

	log.Print("Marking this server as ready")
	err = s.Ready()
	if err != nil {
		log.Fatalf("Could not send ready message")
	}

	err = s.WatchGameServer(readGameServerMeta)
	if err != nil {
		log.Fatalf("Could not watch for game server configuration changes")
	}

	err = runServer(ctx, ":10001") // Bind to all IPs for the machine
	if err != nil {
		log.Print(err)
	}
}

func readGameServerMeta(gs *sdk.GameServer) {
	log.Println("Watching GameServer configuration update")
	if meta := gs.ObjectMeta; meta != nil {
		state.gsName = meta.Name
		state.matchID = meta.Labels["openamtch/match"]

		if j, ok := meta.Annotations["openmatch/rosters"]; ok {
			playersRosters, err := unmarshalRosters(j)
			if err != nil {
				log.Printf("Could not unmarshall the value of \"%s\" annotation: `%s`", "openmatch/rosters", j)
			}
			numPlayersChanged := len(state.playersRosters) != len(playersRosters)
			if numPlayersChanged {
				state.playersRosters = playersRosters
				log.Printf("Read players & rosters: %+v", state.playersRosters)
			}
		}
	}
}

func unmarshalRosters(j string) (map[string]string, error) {
	var rosters []struct {
		Name    string `json:"name,omitempty"`
		Players []struct {
			ID string `json:"id,omitempty"`
		} `json:"players,omitempty"`
	}

	err := json.Unmarshal([]byte(j), &rosters)
	if err != nil {
		return nil, err
	}

	playersRosters := make(map[string]string)
	for _, r := range rosters {
		for _, p := range r.Players {
			if p.ID != "" {
				playersRosters[p.ID] = r.Name
			}
		}
	}
	return playersRosters, nil
}

func runServer(ctx context.Context, address string) error {
	// Start listening
	log.Printf("Listening on %v", address)
	pc, err := net.ListenPacket("udp", address)
	if err != nil {
		return err
	}
	defer pc.Close()

	s := newServer(pc)
	errC := make(chan error)

	var wg sync.WaitGroup
	wg.Add(2)

	// Run commands loop handler
	go func() {
		defer wg.Done()
		err = doREPL(s)
		if err != nil {
			errC <- fmt.Errorf("Error from REPL: " + err.Error())
		}
	}()

	// Run timestamps broadcasting process
	brDone := make(chan struct{})
	go func() {
		defer wg.Done()
		tick := time.Tick(1 * time.Second)
		for {
			select {
			case <-brDone:
				return
			case t := <-tick:
				if len(s.subscribers) > 0 {
					err := s.broadcast(fmt.Sprintf("%v", t.Unix()))
					if err != nil {
						errC <- fmt.Errorf("Error from broadcaster: " + err.Error())
						return
					}
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
	case <-s.exitC:
	case err = <-errC:
	}

	close(brDone)
	s.dying = true
	pc.Close()
	wg.Wait()

	return err
}

func doREPL(s *server) error {
	helpText := strings.Join([]string{
		"Welcome to DGS \"" + state.gsName + "\"",
		"This is a simple UDP server to which you can send some commands:",
		"    `<playerID> START`: to subscribe to broadcasts;",
		"    `<playerID> STOP`:  to unsubscribe from broadcasts;",
		"    `<playerID> EXIT`:  to ask the server to exit;",
		"    `HELP`:             to get this help text.",
	}, "\n")

	buf := make([]byte, maxBufferSize)
	for {
		addr, msg, err := s.read(buf)
		if s.dying {
			return nil
		}
		if err != nil {
			return err
		}

		if "HELP" == strings.ToUpper(msg) {
			_, err = s.pc.WriteTo([]byte(helpText), addr)
			if err != nil {
				return err
			}
			continue
		}

		// Clients expected to send message in format <PLAYER_ID COMMAND>
		words := strings.Split(msg, " ")
		if len(words) < 2 {
			continue
		}
		playerID, cmd := words[0], strings.ToUpper(words[1])

		// Ignore unknown players
		if _, ok := state.playersRosters[playerID]; !ok {
			log.Printf("Unauthorized connection attempt from %s: `%s`", addr.String(), msg)
			continue
		}

		switch cmd {
		case "START":
			err := s.broadcast(fmt.Sprintf("Player %s connected from %s", playerID, addr.String()))
			if err != nil {
				return err
			}
			s.subscribe(playerID, &addr)
			if err = s.send(playerID, fmt.Sprintf("Hello, %s, you've subscribed to broadcasts", playerID)); err != nil {
				return err
			}

		case "STOP":
			err := s.send(playerID, fmt.Sprintf("Bye, %s, you've unsubscribed", playerID))
			if err != nil {
				return err
			}
			s.unsubscribe(playerID)
			if err = s.broadcast(fmt.Sprintf("Player %s disconnected", playerID)); err != nil {
				return err
			}

		case "EXIT":
			err := s.broadcast(fmt.Sprintf("Player %s asked server to shutdown", playerID))
			if err != nil {
				return err
			}
			s.exit()

		case "HELP":
			_, err = s.pc.WriteTo([]byte(helpText), addr)
			if err != nil {
				return err
			}

		default:
		}
	}
}

// doHealth sends the regular Health Pings
func doHealth(s *gosdk.SDK, stop <-chan struct{}) {
	tick := time.Tick(2 * time.Second)
	for {
		err := s.Health()
		if err != nil {
			log.Fatalf("Could not send health ping, %v", err)
		}
		select {
		case <-stop:
			log.Print("Stopped health pings")
			return
		case <-tick:
		}
	}
}
