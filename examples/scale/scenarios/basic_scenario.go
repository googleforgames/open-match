package scenarios

import (
	"fmt"
	"open-match.dev/open-match/pkg/pb"
)

// BasicScenario is a struct with Scenario embedded that implements mmf.Run and evaluator.Evaluate methods.
type BasicScenario struct {
	Scenario
}

// Run implements the basic matchfunction.Run logic for Open Match benchmarking under BasicScenario
func (s BasicScenario) Run(*pb.RunRequest, pb.MatchFunction_RunServer) error {
	fmt.Println("hello")
	return nil
}

// Evaluate implements the basic evaluator.Evaluate logic for Open Match benchmarking under BasicScenario
func (s BasicScenario) Evaluate(pb.Evaluator_EvaluateServer) error {
	fmt.Println("hi")
	return nil
}
