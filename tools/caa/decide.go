package main

import "strings"

// Event is the part of the GitHub event payload the decision depends on. It is
// filled from the event file and one read of the pull request, and nothing in
// the decision below reaches back to the API, so the whole policy is testable
// as a pure function.
type Event struct {
	Name             string // pull_request_target or issue_comment
	OnPullRequest    bool   // an issue_comment is on a pull request, not a plain issue
	RepoID           int64
	PullRequestNo    int
	HeadSHA          string
	Author           string // the pull request's author
	CommentBody      string
	CommentAuthor    string
	CommentAuthorID  int64
	CommentID        int64
	CommentCreatedAt string
}

// Config is the policy the workflow passes in.
type Config struct {
	Sentence  string   // the exact sentence a signature must be
	Allowlist []string // logins that never need to sign
}

// Decision is what the run does. Nothing else in the program decides anything.
type Decision struct {
	Skip      bool // the event is not about a pull request: do nothing at all
	Sign      bool // append Signature to the ledger
	Signature Signature
	Comment   bool   // ask the author to sign, unless the ask is already on the thread
	Status    string // success or failure, always set when Skip is false
}

// isBot reports whether a login is a GitHub App account. Bots author commits
// and comments in our own automation and never sign.
func isBot(login string) bool {
	return strings.HasSuffix(login, "[bot]")
}

func allowlisted(login string, allowlist []string) bool {
	for _, allowed := range allowlist {
		if allowed != "" && strings.EqualFold(allowed, login) {
			return true
		}
	}
	return false
}

// IsSignature reports whether a comment body is the signature sentence. The
// whole body must be the sentence: leading and trailing whitespace is ignored
// and the comparison is case insensitive, but a body carrying any other text is
// not a signature.
func IsSignature(body, sentence string) bool {
	return strings.EqualFold(strings.TrimSpace(body), strings.TrimSpace(sentence))
}

// Decide is the entire policy.
func Decide(ev Event, led *Ledger, cfg Config) Decision {
	if ev.Name == "issue_comment" && !ev.OnPullRequest {
		return Decision{Skip: true}
	}

	if allowlisted(ev.Author, cfg.Allowlist) || isBot(ev.Author) {
		return Decision{Status: "success"}
	}

	dec := Decision{}
	signed := led.Has(ev.Author)

	// A signature counts only when the pull request's own author posts it. A
	// comment carrying the sentence from anyone else signs nothing.
	if !signed && ev.Name == "issue_comment" &&
		strings.EqualFold(ev.CommentAuthor, ev.Author) &&
		IsSignature(ev.CommentBody, cfg.Sentence) {
		dec.Sign = true
		dec.Signature = Signature{
			Name:          ev.CommentAuthor,
			ID:            ev.CommentAuthorID,
			CommentID:     ev.CommentID,
			CreatedAt:     ev.CommentCreatedAt,
			RepoID:        ev.RepoID,
			PullRequestNo: ev.PullRequestNo,
		}
		signed = true
	}

	if signed {
		dec.Status = "success"
		return dec
	}

	dec.Status = "failure"
	// The ask goes out when the pull request opens or moves. A comment that is
	// not a signature restates the status and says nothing on the thread.
	dec.Comment = ev.Name == "pull_request_target"
	return dec
}
