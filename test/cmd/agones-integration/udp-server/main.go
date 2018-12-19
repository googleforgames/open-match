package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"agones.dev/agones/pkg/util/signals"
	sdk "agones.dev/agones/sdks/go"
)

const maxBufferSize = 1024

var clients = make([]*net.Addr, 0)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		stop := signals.NewStopChannel()
		<-stop
		log.Println("Exit signal received, stopping everything")
		cancel()
	}()

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
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

	err = server(ctx, ":10001") // Bind to all IPs for the machine
	if err != nil {
		log.Print(err)
	}
}

func server(ctx context.Context, address string) (err error) {
	// Start listening
	log.Printf("Listening on %v:%v", address)
	pc, err := net.ListenPacket("udp", address)
	if err != nil {
		return
	}
	defer pc.Close()

	stop := make(chan struct{})
	errChan := make(chan error, 2)
	buffer := make([]byte, maxBufferSize)

	// Wait for new clients, add to the list
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			default:
				_, addr, err := pc.ReadFrom(buffer)
				if err != nil {
					errChan <- fmt.Errorf("Error reading: %s", err)
					return
				}
				clients = append(clients, &addr)
				log.Printf("New client %s\n", addr.String())
			}
		}
	}()

	// Loop forever, sending timestamp to all clients in the list.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			default:
				ts := time.Now().Unix()
				b := []byte(fmt.Sprintf("%v", ts))
				log.Printf("Sending %v\n", ts)
				for _, addr := range clients {
					_, err = pc.WriteTo(b, *addr)
					if err != nil {
						errChan <- fmt.Errorf("Error sending: %s", err)
						return
					}
				}
				time.Sleep(1 * time.Second)
			}
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("context cancelled")
		// err = ctx.Err()
	case err = <-errChan:
	}

	close(stop)
	return
}

// doHealth sends the regular Health Pings
func doHealth(s *sdk.SDK, stop <-chan struct{}) {
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
