package model

// Stats is a compact report of one service's registry state.
type Stats struct {
	Service   string
	Instances int
	Healthy   int
	Suspect   int
	Evicted   int
}
