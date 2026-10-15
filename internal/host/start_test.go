package host

import (
	"testing"
	"time"
)

func TestWaitStartedWaitsForTheHolder(t *testing.T) {
	alertHome(t)
	m := &Manifest{Schema: 1, ID: NewID(), Status: Status{State: StateStarting}}
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(150 * time.Millisecond)
		mm := *m
		mm.Run.AgentPID, mm.Status.State = 4242, StateWorking
		_ = mm.Save()
	}()
	got := waitStarted(m.ID, 5*time.Second)
	if got == nil || got.Run.AgentPID != 4242 {
		t.Fatalf("waitStarted returned %+v before the holder wrote its start", got)
	}
	if got := waitStarted("nosuchsession", 50*time.Millisecond); got != nil {
		t.Fatalf("a missing session gave %+v", got)
	}
}
