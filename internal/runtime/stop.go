package runtime

import "fmt"

type StopReason string

const (
	StopEndTurn    StopReason = "end_turn"
	StopMaxTurns   StopReason = "max_turns"
	StopMaxTokens  StopReason = "max_tokens"
	StopBudget     StopReason = "budget"
	StopCancelled  StopReason = "cancelled"
	StopOverflow   StopReason = "overflow"
	StopHook       StopReason = "hook_stop"
	StopModelError StopReason = "model_error"
)

type StopError struct {
	Reason StopReason
	Msg    string
	Err    error
}

func (e *StopError) Error() string {
	if e == nil {
		return ""
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Reason)
}

func (e *StopError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func stopErr(reason StopReason, msg string, err error) error {
	return &StopError{Reason: reason, Msg: msg, Err: err}
}

func ReasonOf(err error) StopReason {
	if err == nil {
		return StopEndTurn
	}
	if e, ok := err.(*StopError); ok {
		return e.Reason
	}
	return StopModelError
}

func maxTurnsErr() error {
	return stopErr(StopMaxTurns, "max turns reached", fmt.Errorf("max turns reached"))
}
