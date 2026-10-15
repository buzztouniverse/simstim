package client

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/iAmLakshya/simstim/internal/api"
)

func TestSetupTrustIsKeptByHostAndRepository(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if got := SetupTrust("box"); len(got) != 0 {
		t.Fatalf("fresh: %v", got)
	}
	if err := RememberSetup("box", "/home/me/src/app", true); err != nil {
		t.Fatal(err)
	}
	if err := RememberSetup("box", "/home/me/src/api", false); err != nil {
		t.Fatal(err)
	}
	if err := RememberSetup("other", "/home/me/src/app", false); err != nil {
		t.Fatal(err)
	}
	got := SetupTrust("box")
	if run, ok := got["/home/me/src/app"]; !ok || !run {
		t.Errorf("app: %v", got)
	}
	if run, ok := got["/home/me/src/api"]; !ok || run {
		t.Errorf("api: %v", got)
	}
	if run := SetupTrust("other")["/home/me/src/app"]; run {
		t.Error("an answer leaked to another host")
	}
	if st, err := os.Stat(trustPath()); err != nil || st.Mode().Perm() != 0o600 {
		t.Errorf("the answers file: %v %v", st, err)
	}
}

func TestSetupQuestionOf(t *testing.T) {
	data, _ := json.Marshal(api.SetupQuestion{Repo: "/r", Script: "/r/s"})
	q, ok := SetupQuestionOf(&UserError{Msg: "m", Code: api.CodeSetupTrust, Data: data})
	if !ok || q.Repo != "/r" || q.Script != "/r/s" {
		t.Fatalf("question %+v %v", q, ok)
	}
	for _, err := range []error{nil, errors.New("x"), &UserError{Msg: "m", Code: "other"}, &UserError{Code: api.CodeSetupTrust}} {
		if _, ok := SetupQuestionOf(err); ok {
			t.Errorf("%v is not a question", err)
		}
	}
}
