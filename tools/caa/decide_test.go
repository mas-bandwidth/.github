package main

import "testing"

const sentence = "I have read the CAA and I hereby sign it, assigning copyright in my contributions to Más Bandwidth LLC."

func testConfig() Config {
	return Config{Sentence: sentence, Allowlist: []string{"gafferongames", "rowan-claude"}}
}

func TestIsSignature(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"the sentence", sentence, true},
		{"surrounding whitespace", "  \n" + sentence + "\n\n", true},
		{"a trailing carriage return", sentence + "\r\n", true},
		{"different capitalization", "i have read the caa and i hereby sign it, assigning copyright in my contributions to más bandwidth llc.", true},
		{"a version pinned remark after it", sentence + " (CAA as of 2026-07-24)", false},
		{"a footnote before it", "Happy to help. " + sentence, false},
		{"the wrong wording", "I have read the CLA Document and I hereby sign the CLA", false},
		{"an empty body", "", false},
		{"the ask to sign, which quotes the sentence", askToSign(sentence), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSignature(tc.body, sentence); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func prEvent(author string) Event {
	return Event{
		Name:          "pull_request_target",
		OnPullRequest: true,
		RepoID:        59925747,
		PullRequestNo: 331,
		HeadSHA:       "864046f9a0fa1d98ec39a57b08bd997a5a3e2e7d",
		Author:        author,
	}
}

func commentEvent(author, commenter, body string) Event {
	ev := prEvent(author)
	ev.Name = "issue_comment"
	ev.CommentBody = body
	ev.CommentAuthor = commenter
	ev.CommentAuthorID = 2938071
	ev.CommentID = 5385662641
	ev.CommentCreatedAt = "2026-08-23T11:04:08Z"
	return ev
}

func TestPullRequestFromAnUnsignedAuthorAsksAndFails(t *testing.T) {
	got := Decide(prEvent("newcomer"), &Ledger{}, testConfig())
	if !got.Comment {
		t.Error("Comment: got false, want true")
	}
	if got.Sign {
		t.Error("Sign: got true, want false")
	}
	if got.Status != "failure" {
		t.Errorf("Status: got %q, want failure", got.Status)
	}
}

func TestPullRequestFromASignedAuthorPassesQuietly(t *testing.T) {
	led := &Ledger{SignedContributors: []Signature{{Name: "Green-Sky"}}}
	got := Decide(prEvent("green-sky"), led, testConfig())
	if got.Comment || got.Sign {
		t.Errorf("got %+v, want no comment and no signature", got)
	}
	if got.Status != "success" {
		t.Errorf("Status: got %q, want success", got.Status)
	}
}

func TestAllowlistedAndBotAuthorsNeverSign(t *testing.T) {
	for _, author := range []string{"gafferongames", "rowan-claude", "GafferOnGames", "dependabot[bot]"} {
		got := Decide(prEvent(author), &Ledger{}, testConfig())
		if got.Status != "success" || got.Comment || got.Sign {
			t.Errorf("%s: got %+v, want a quiet success", author, got)
		}
	}
}

func TestTheAuthorSigningRecordsTheSignature(t *testing.T) {
	got := Decide(commentEvent("Green-Sky", "Green-Sky", sentence), &Ledger{}, testConfig())
	if !got.Sign {
		t.Fatal("Sign: got false, want true")
	}
	if got.Status != "success" {
		t.Errorf("Status: got %q, want success", got.Status)
	}
	if got.Comment {
		t.Error("Comment: got true, want false: a signature is answered by the status, not by another comment")
	}
	want := Signature{
		Name:          "Green-Sky",
		ID:            2938071,
		CommentID:     5385662641,
		CreatedAt:     "2026-08-23T11:04:08Z",
		RepoID:        59925747,
		PullRequestNo: 331,
	}
	if got.Signature != want {
		t.Fatalf("Signature: got %+v, want %+v", got.Signature, want)
	}
}

// The negative control: the sentence carries weight only from the person whose
// contribution it assigns. Anyone else posting it signs nothing, and leaves the
// pull request red.
func TestTheSentenceFromAnotherUserSignsNothing(t *testing.T) {
	got := Decide(commentEvent("newcomer", "bystander", sentence), &Ledger{}, testConfig())
	if got.Sign {
		t.Fatalf("Sign: got true, want false: %+v", got.Signature)
	}
	if got.Status != "failure" {
		t.Errorf("Status: got %q, want failure", got.Status)
	}
}

// A signed contributor commenting the sentence on someone else's pull request
// must not carry that pull request.
func TestASignedBystanderDoesNotCarryTheAuthor(t *testing.T) {
	led := &Ledger{SignedContributors: []Signature{{Name: "bystander"}}}
	got := Decide(commentEvent("newcomer", "bystander", sentence), led, testConfig())
	if got.Sign || got.Status != "failure" {
		t.Fatalf("got %+v, want no signature and a failure", got)
	}
}

func TestACommentThatIsNotTheSentenceSignsNothing(t *testing.T) {
	for _, body := range []string{"recheck", "LGTM", sentence + " -- with reservations"} {
		got := Decide(commentEvent("newcomer", "newcomer", body), &Ledger{}, testConfig())
		if got.Sign {
			t.Errorf("%q: signed, want no signature", body)
		}
		if got.Status != "failure" {
			t.Errorf("%q: Status got %q, want failure", body, got.Status)
		}
		if got.Comment {
			t.Errorf("%q: Comment got true, want false", body)
		}
	}
}

// Signing twice records once. The second run reads a ledger that already holds
// the author and restates the passing status.
func TestSigningAnAlreadySignedAuthorRecordsNothing(t *testing.T) {
	led := &Ledger{SignedContributors: []Signature{{Name: "Green-Sky", ID: 2938071, PullRequestNo: 331}}}
	got := Decide(commentEvent("Green-Sky", "Green-Sky", sentence), led, testConfig())
	if got.Sign {
		t.Fatal("Sign: got true, want false")
	}
	if got.Status != "success" {
		t.Errorf("Status: got %q, want success", got.Status)
	}
	if len(led.SignedContributors) != 1 {
		t.Fatalf("ledger grew to %d entries", len(led.SignedContributors))
	}
}

// A comment on a plain issue is not a pull request event. The run does nothing
// at all: no status, because there is no commit to set one on.
func TestACommentOnAnIssueIsSkipped(t *testing.T) {
	ev := commentEvent("newcomer", "newcomer", sentence)
	ev.OnPullRequest = false
	got := Decide(ev, &Ledger{}, testConfig())
	if !got.Skip {
		t.Fatalf("Skip: got false, want true: %+v", got)
	}
	if got.Status != "" || got.Sign || got.Comment {
		t.Fatalf("got %+v, want an empty decision", got)
	}
}
