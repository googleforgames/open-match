package agones

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"github.com/GoogleCloudPlatform/open-match/internal/pb"
)

// GameServerAllocator allocates game servers in Agones fleet
type GameServerAllocator struct {
	agonesClient *versioned.Clientset

	namespace    string
	fleetName    string
	generateName string

	l *logrus.Entry
}

// NewAllocator creates new GameServerAllocator with in cluster k8s config
func NewGameServerAllocator(namespace, fleetName, generateName string, l *logrus.Entry) (*GameServerAllocator, error) {
	agonesClient, err := getAgonesClient()
	if err != nil {
		return nil, errors.New("Could not create Agones game server allocator: " + err.Error())
	}

	a := &GameServerAllocator{
		agonesClient: agonesClient,

		namespace:    namespace,
		fleetName:    fleetName,
		generateName: generateName,

		l: l.WithFields(logrus.Fields{
			"source":       "agones",
			"namespace":    namespace,
			"fleetname":    fleetName,
			"generatename": generateName,
		}),
	}
	return a, nil
}

// Set up our client which we will use to call the API
func getAgonesClient() (*versioned.Clientset, error) {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.New("Could not create in cluster config: " + err.Error())
	}

	// Access to the Agones resources through the Agones Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, errors.New("Could not create the agones api clientset: " + err.Error())
	}
	return agonesClient, nil
}

// Allocate allocates a game server in a fleet, distributes match object details to it,
// and returns a connection string or error
func (a *GameServerAllocator) Allocate(match *pb.MatchObject) (string, error) {

	labels, annotations := a.getAllocationMeta(match)

	// Define the fleet allocation using the constants set earlier
	faReq := &v1alpha1.FleetAllocation{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: a.generateName, Namespace: a.namespace,
		},
		Spec: v1alpha1.FleetAllocationSpec{
			FleetName: a.fleetName,
			MetaPatch: v1alpha1.MetaPatch{Labels: labels, Annotations: annotations},
		},
	}

	// Create a new fleet allocation
	fa, err := a.agonesClient.StableV1alpha1().FleetAllocations(a.namespace).Create(faReq)
	if err != nil {
		return "", errors.New("Failed to create fleet allocation: " + err.Error())
	}

	dgs := fa.Status.GameServer.Status
	connstring := fmt.Sprintf("%s:%d", dgs.Address, dgs.Ports[0].Port)

	a.l.WithFields(logrus.Fields{
		"fleetallocation": fa.Name,
		"gameserver":      fa.Status.GameServer.Name,
		"connstring":      connstring,
	}).Info("GameServer allocated")

	return connstring, nil
}

func (a *GameServerAllocator) getAllocationMeta(match *pb.MatchObject) (labels, annotations map[string]string) {
	labels = map[string]string{
		"openmatch/match": match.Id,
	}

	annotations = map[string]string{}

	if pools, err := json.Marshal(match.Pools); err == nil {
		annotations["openmatch/pools"] = string(pools)
	} else {
		a.l.WithField("match", match.Id).
			WithError(err).
			Error("Could not marhsal MatchObject.Pools to attach to FleetAllocation metadata")
	}

	// Propagate the rosters using the distinct method Send()

	return
}

// Send updates the metadata of allocated GS at given connstring
func (a *GameServerAllocator) Send(connstring string, match *pb.MatchObject) error {
	gs, err := a.findGameServer(connstring)
	if err != nil {
		return errors.New("could not find game server: " + err.Error())
	}

	gs0 := gs.DeepCopy()
	a.copyNewlyFilledRosters(gs, match)
	patch, err := gs0.Patch(gs)
	if err != nil {
		return errors.New("could not compute JSON patch to update the game server metadata: " + err.Error())
	}

	gsi := a.agonesClient.StableV1alpha1().GameServers(a.namespace)
	_, err = gsi.Patch(gs0.Name, types.JSONPatchType, patch)
	if err != nil {
		return errors.New("could not update game server: " + err.Error())
	}

	a.l.WithField("connstring", connstring).
		WithField("gameserver", gs.Name).
		WithField("match.id", match.Id).
		WithField("patch", string(patch)).
		Info("Data from new MatchObject was sent to game server")
	return nil
}

func (a *GameServerAllocator) copyNewlyFilledRosters(gs *v1alpha1.GameServer, newmatch *pb.MatchObject) {
	var rosters []*pb.Roster

	if curJ, ok := gs.Annotations["openmatch/rosters"]; !ok {
		// If it's missing then most probably it was never set
		rosters = newmatch.Rosters

	} else {
		if err := json.Unmarshal([]byte(curJ), &rosters); err != nil {
			a.l.WithError(err).
				WithField("gs.name", gs.Name).
				WithField("openmatch/rosters", gs.Annotations["openmatch/rosters"]).
				Error("Could not unmarshal the value of rosters annotation")
			return
		}

		// Iterate over newly filled rosters and copy players into empty slots of current rosters
		for _, fromRoster := range newmatch.Rosters {
			// Find matching current roster
			var roster *pb.Roster
			for _, intoRoster := range rosters {
				if intoRoster.Name == fromRoster.Name {
					roster = intoRoster
					break
				}
			}
			if roster != nil {
				// Copy player IDs into empty slots of matching current roster
				for _, p := range fromRoster.Players {
					// Find matching empty slot
					var room *pb.Player
					for _, slot := range roster.Players {
						if slot.Pool == p.Pool && slot.Id == "" {
							room = slot
							break
						}
					}
					if room != nil {
						room.Id = p.Id
					}
				}
			}
		}

	}

	newJ, err := json.Marshal(rosters)
	if err != nil {
		a.l.WithError(err).
			WithField("gs.name", gs.Name).
			WithField("openmatch/rosters", gs.Annotations["openmatch/rosters"]).
			WithField("newmatch", newmatch).
			WithField("rosters", rosters).
			Error("Could not marshal the updated rosters to JSON")
		return
	}

	gs.Annotations["openmatch/rosters"] = string(newJ)
}

// UnAllocate finds and deletes the allocated game server matching the specified connection string
func (a *GameServerAllocator) UnAllocate(connstring string) error {
	gs, err := a.findGameServer(connstring)
	if err != nil {
		return errors.New("could not find game server: " + err.Error())
	}
	if gs == nil {
		return errors.New("found no game servers matching the connection string")
	}

	fields := logrus.Fields{
		"connstring": connstring,
		"gameserver": gs.Name,
	}

	gsi := a.agonesClient.StableV1alpha1().GameServers(a.namespace)
	err = gsi.Delete(gs.Name, nil)
	if err != nil {
		msg := "failed to delete game server"
		a.l.WithFields(fields).WithError(err).Error(msg)
		return errors.New(msg + ": " + err.Error())
	}
	a.l.WithFields(fields).Info("GameServer deleted")

	return nil
}

func (a *GameServerAllocator) findGameServer(connstring string) (*v1alpha1.GameServer, error) {
	var ip, port string
	if parts := strings.Split(connstring, ":"); len(parts) != 2 {
		return nil, errors.New("unable to parse connection string: expecting format \"<IP>:<PORT>\"")
	} else {
		ip, port = parts[0], parts[1]
	}

	gsi := a.agonesClient.StableV1alpha1().GameServers(a.namespace)

	gsl, err := gsi.List(v1.ListOptions{})
	if err != nil {
		return nil, errors.New("failed to get game servers list: " + err.Error())
	}

	for _, gs := range gsl.Items {
		if /*gs.Status.State == "Allocated" &&*/ gs.Status.Address == ip {
			for _, p := range gs.Status.Ports {
				if strconv.Itoa(int(p.Port)) == port {
					return &gs, nil
				}
			}
		}
	}
	return nil, nil
}
