/*
Evaluator harness contains common scaffolding needed by the Evaluator. The
harness currently runs the evaluator loop that periodically scans the database
for new proposals and calls the user defined evaluation function with the
collection of proposals. The harness accepts the approved matches from the user
and modifies the database to indicate these results.

Currently, the harness logic simply abstracts the user from the database
concepts. In future, as the proposals move out of the database, this harness
will be a GRPC service that accepts proposal stream from Open Match and
calls user's evaluation logic with this collection of proposals. Thus the
harness will minimize impact of these future changes on user's evaluation
function.
*/

package harness
