package model

import apperror "github.com/farid/reservation-service/pkg/error"

// Action names a transition trigger.
type Action string

const (
	ActionConfirm  Action = "CONFIRM"
	ActionCancel   Action = "CANCEL"
	ActionCheckIn  Action = "CHECK_IN"
	ActionCheckOut Action = "CHECK_OUT"
	ActionExpire   Action = "EXPIRE" // worker-only
)

var allowed = map[ReservationState]map[Action]ReservationState{
	StatePending: {
		ActionConfirm: StateConfirmed,
		ActionCancel:  StateCancelled,
	},
	StateConfirmed: {
		ActionCheckIn: StateActive,
		ActionCancel:  StateCancelled,
		ActionExpire:  StateExpired,
	},
	StateActive: {
		ActionCheckOut: StateCompleted,
		ActionCancel:   StateCancelled,
	},
}

// Next returns the next state for (from, action), or ErrInvalidState if illegal.
func Next(from ReservationState, action Action) (ReservationState, error) {
	transitions, ok := allowed[from]
	if !ok {
		return "", apperror.ErrInvalidState
	}
	to, ok := transitions[action]
	if !ok {
		return "", apperror.ErrInvalidState
	}
	return to, nil
}
