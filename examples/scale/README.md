## How to use this framework

This is the framework that we use to benchmark Open Match against different matchmaking scenarios. For now (02/24/2020), this framework supports a Battle Royale, a Basic 1v1 matchmaking, and a Team Shooter scenario. You are welcome to write up your own `Scenario`, test it, and share the number that you are able to get to us. 

1. The `Scenario` struct under the `scenarios/scenarios.go` file defines the parameters that this framework currently support/plan to support.
2. Each subpackage `battleroyal`, `firstmatch`, and `teamshooter` implements to `GameScenario` interface defined under `scenarios/scenarios.go` file. Feel free to write your own benchmark scenario by implementing the interface. 
   - Ticket   `func() *pb.Ticket` - Tickets generator
   - Profiles `func() []*pb.MatchProfile` - Profiles generator
   - MMF      `MatchFunction(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error)` - Custom matchmaking logic using a MatchProfile and a map struct that contains the mapping from pool name to the tickets of that pool.
   - Evaluate `Evaluate(stream pb.Evaluator_EvaluateServer) error` - Custom logic implementation of the evaluator.

Follow the instructions below if you want to use any of the existing benchmarking scenarios.

1. Open the `scenarios.go` file under the scenarios directory.
2. Change the value of the `ActiveScenario` variable to the scenario that you would like Open Match to run against.
3. Make sure you have `kubectl` connected to an existing Kubernetes cluster and run `make push-images` followed by `make install-scale-chart` to push the images and install Open Match core along with the scale components in the cluster.
4. Run `make proxy` 
   - Open `localhost:3000` to see the Grafana dashboards.
   - Open `localhost:9090` to see the Prometheus query server.
   - Open `localhost:[COMPONENT_HTTP_ENDPOINT]/help` to see how to access the zpages.
