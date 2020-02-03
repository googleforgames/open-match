## How to use this framework

This is the framework that we use to benchmark Open Match against different matchmaking scenarios. The `Scenario` struct under the `scenarios/scenarios.go` file defines the parameters that this framework currently support/plan to support. For now (02/01/2020), this framework supports a Battle Royale and a Basic 1v1 matchmaking scenario. You are welcome to write up your own `Scenario`, test it, and share the number that you are able to get to us. 

Follow the instructions below if you want to use any of the existing benchmarking scenarios.

1. Open the `scenarios.go` file under the scenarios directory.
2. Change the value of the `ActiveScenario` variable to the scenario that you would like Open Match to run against.
3. Make sure you have `kubectl` connected to an existing Kubernetes cluster and run `make push-images` followed by `make install-scale-chart` to push the images and install Open Match core along with the scale components in the cluster.
4. Run `make proxy` 
   - Open `localhost:3000` to see the Grafana dashboards.
   - Open `localhost:9090` to see the Prometheus query server.
   - Open `localhost:[COMPONENT_HTTP_ENDPOINT]/help` to see how to access the zpages.
