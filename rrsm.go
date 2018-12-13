package rrsm

// interface used to abstract a state of RRSM
type RRSMState interface {
	// return the state name
	Info() string

	// return next state
	Handle(command string) interface{}
}
