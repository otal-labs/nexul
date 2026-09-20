package deploy

// allowedTransitions is the deploy state machine; terminal states have no outbound edges.
var allowedTransitions = map[Status]map[Status]bool{
	StatusPending: {StatusHealthy: true, StatusFailed: true},
	StatusRunning: {StatusHealthy: true, StatusFailed: true},
}

// CanTransition reports whether to is reachable from from in one step.
func CanTransition(from, to Status) bool {
	next, ok := allowedTransitions[from]
	return ok && next[to]
}
