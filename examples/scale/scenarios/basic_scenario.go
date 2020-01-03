package scenarios

import "open-match.dev/open-match/pkg/pb"

var basicScenario = &Scenario{
	MMF:       basicMatchFunction,
	Evaluator: basicEvaluate,
}

func basicMatchFunction(*pb.RunRequest, pb.MatchFunction_RunServer) error {
	return nil
}

func basicEvaluate(pb.Evaluator_EvaluateServer) error {
	return nil
}
