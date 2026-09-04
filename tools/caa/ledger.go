package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Signature is one entry in the signature ledger. The field names and the
// order they marshal in are the shape the ledger already holds, so entries
// written before this program stay valid and a round trip is byte for byte.
type Signature struct {
	Name          string `json:"name"`
	ID            int64  `json:"id"`
	CommentID     int64  `json:"comment_id"`
	CreatedAt     string `json:"created_at"`
	RepoID        int64  `json:"repoId"`
	PullRequestNo int    `json:"pullRequestNo"`
}

// Ledger is the whole signature file: signatures/caa.json on the
// cla-signatures branch of mas-bandwidth/.github.
type Ledger struct {
	SignedContributors []Signature `json:"signedContributors"`
}

// ParseLedger reads the ledger file. An empty file is an empty ledger, so a
// ledger that does not exist yet does not need special handling upstream.
func ParseLedger(raw []byte) (*Ledger, error) {
	led := &Ledger{}
	if len(bytes.TrimSpace(raw)) == 0 {
		return led, nil
	}
	if err := json.Unmarshal(raw, led); err != nil {
		return nil, fmt.Errorf("parse ledger: %w", err)
	}
	return led, nil
}

// Marshal renders the ledger in the file's existing formatting: two space
// indent, one trailing newline.
func (l *Ledger) Marshal() ([]byte, error) {
	raw, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render ledger: %w", err)
	}
	return append(raw, '\n'), nil
}

// Has reports whether a login already appears in the ledger. GitHub logins are
// case insensitive, so the comparison is too.
func (l *Ledger) Has(login string) bool {
	for _, sig := range l.SignedContributors {
		if strings.EqualFold(sig.Name, login) {
			return true
		}
	}
	return false
}

// Append adds a signature and reports whether the ledger changed. Appending a
// login the ledger already holds changes nothing, so a replayed or duplicated
// event cannot write a second entry for the same person.
func (l *Ledger) Append(sig Signature) bool {
	if l.Has(sig.Name) {
		return false
	}
	l.SignedContributors = append(l.SignedContributors, sig)
	return true
}
