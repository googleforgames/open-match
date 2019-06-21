package synchronizer

import (
	"open-match.dev/open-match/pkg/pb"
)

// EvaluatorFunction is the signature of the function that the synchronizer
// service invokes for evaluation.
type EvaluatorFunction func([]*pb.Match) []*pb.Match

func evaluate([]*pb.Match) []*pb.Match {
	// TODO: Add the logic to trigger HTTP / GRPC evaluation client based on user
	// configuration to trigger the custom evaluator here.
	// Currently, this hook is only used for unit testing the synchronizer.
	return nil
}

func getEvaluator() EvaluatorFunction {
	return evaluate
}
