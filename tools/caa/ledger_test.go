package main

import (
	"bytes"
	"os"
	"testing"
)

// testdata/ledger.json is a snapshot of signatures/caa.json as it stands on the
// cla-signatures branch, written by the automation this program replaces.
const snapshot = "testdata/ledger.json"

func TestParseLedgerReadsTheExistingFile(t *testing.T) {
	raw, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	led, err := ParseLedger(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(led.SignedContributors) != 21 {
		t.Fatalf("signatures: got %d, want 21", len(led.SignedContributors))
	}

	first := led.SignedContributors[0]
	want := Signature{
		Name:          "bingham909",
		ID:            239700075,
		CommentID:     5018843853,
		CreatedAt:     "2026-07-20T04:38:33Z",
		RepoID:        59925747,
		PullRequestNo: 307,
	}
	if first != want {
		t.Fatalf("first signature: got %+v, want %+v", first, want)
	}
}

// A signature recorded before this program has to keep counting, and the file
// this program writes has to stay the file the ledger already is, byte for
// byte, so a diff shows only the appended entry.
func TestLedgerRoundTripIsByteIdentical(t *testing.T) {
	raw, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	led, err := ParseLedger(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := led.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !bytes.Equal(raw, out) {
		t.Fatalf("round trip changed the file:\n--- got ---\n%s\n--- want ---\n%s", out, raw)
	}
}

func TestParseLedgerAcceptsAnEmptyFile(t *testing.T) {
	for _, raw := range [][]byte{nil, []byte(""), []byte("  \n")} {
		led, err := ParseLedger(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		if len(led.SignedContributors) != 0 {
			t.Fatalf("parse %q: got %d signatures, want 0", raw, len(led.SignedContributors))
		}
	}
}

func TestParseLedgerRejectsGarbage(t *testing.T) {
	if _, err := ParseLedger([]byte("not json")); err == nil {
		t.Fatal("parsing garbage returned no error")
	}
}

func TestHasIsCaseInsensitive(t *testing.T) {
	led := &Ledger{SignedContributors: []Signature{{Name: "Green-Sky"}}}
	for _, login := range []string{"Green-Sky", "green-sky", "GREEN-SKY"} {
		if !led.Has(login) {
			t.Fatalf("Has(%q): got false, want true", login)
		}
	}
	if led.Has("green-skye") {
		t.Fatal("Has(green-skye): got true, want false")
	}
}

func TestAppendIsIdempotent(t *testing.T) {
	led := &Ledger{}
	sig := Signature{Name: "octocat", ID: 1, CommentID: 2, CreatedAt: "2026-09-04T00:00:00Z", RepoID: 3, PullRequestNo: 4}

	if !led.Append(sig) {
		t.Fatal("first append: got false, want true")
	}
	if led.Append(sig) {
		t.Fatal("second append: got true, want false")
	}
	// A second run carrying a different comment on the same login is still the
	// same person, and still must not add a row.
	other := sig
	other.CommentID = 99
	other.PullRequestNo = 42
	if led.Append(other) {
		t.Fatal("append of the same login from another comment: got true, want false")
	}
	// A different login does append.
	if !led.Append(Signature{Name: "hubot", ID: 5}) {
		t.Fatal("append of a new login: got false, want true")
	}
	if len(led.SignedContributors) != 2 {
		t.Fatalf("signatures: got %d, want 2", len(led.SignedContributors))
	}
}
