package model

import (
	"testing"

	apperror "github.com/farid/reservation-service/pkg/error"
)

func TestNext_AllowedTransitions(t *testing.T) {
	cases := []struct {
		from   ReservationState
		action Action
		want   ReservationState
	}{
		{StatePending, ActionConfirm, StateConfirmed},
		{StatePending, ActionCancel, StateCancelled},
		{StateConfirmed, ActionCheckIn, StateActive},
		{StateConfirmed, ActionCancel, StateCancelled},
		{StateConfirmed, ActionExpire, StateExpired},
		{StateActive, ActionCheckOut, StateCompleted},
		{StateActive, ActionCancel, StateCancelled},
	}
	for _, tc := range cases {
		got, err := Next(tc.from, tc.action)
		if err != nil {
			t.Errorf("%s+%s: unexpected error %v", tc.from, tc.action, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s+%s: got %s, want %s", tc.from, tc.action, got, tc.want)
		}
	}
}

func TestNext_IllegalTransitions(t *testing.T) {
	cases := []struct {
		from   ReservationState
		action Action
	}{
		{StateCompleted, ActionCancel},
		{StateCancelled, ActionConfirm},
		{StateExpired, ActionCheckIn},
		{StatePending, ActionCheckIn},  // can't check-in before confirm
		{StatePending, ActionCheckOut}, // can't check-out before active
	}
	for _, tc := range cases {
		_, err := Next(tc.from, tc.action)
		if !apperror.Is(err, apperror.ErrInvalidState) {
			t.Errorf("%s+%s: expected ErrInvalidState, got %v", tc.from, tc.action, err)
		}
	}
}
