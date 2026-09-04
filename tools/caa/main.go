// Command caa gates a pull request on the Contributor Assignment Agreement.
//
// It runs from .github/workflows/caa.yml in this repository, which every
// mas-bandwidth library calls as a reusable workflow. On a pull request whose
// author has not signed it posts the ask to sign; on a comment that is the
// signature sentence, from the pull request's own author, it appends the
// signature to the ledger. Every run ends by setting the commit status that
// branch protection watches.
//
// Everything it needs arrives in the environment. Nothing from an event
// payload is ever interpolated into a command line.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// marker identifies the ask to sign so it is posted once per pull request.
const marker = "<!-- mas-bandwidth-caa -->"

type payload struct {
	Repository struct {
		ID       int64  `json:"id"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest *struct {
		Number int `json:"number"`
		User   struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	} `json:"pull_request"`
	Issue *struct {
		Number      int             `json:"number"`
		PullRequest json.RawMessage `json:"pull_request"`
		User        struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"issue"`
	Comment *struct {
		ID        int64  `json:"id"`
		Body      string `json:"body"`
		CreatedAt string `json:"created_at"`
		User      struct {
			Login string `json:"login"`
			ID    int64  `json:"id"`
		} `json:"user"`
	} `json:"comment"`
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func must(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		fail("%s is not set", name)
	}
	return value
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "caa: "+format+"\n", args...)
	os.Exit(1)
}

func splitList(raw string) []string {
	var out []string
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func main() {
	eventName := must("CAA_EVENT_NAME")
	repo := must("CAA_REPOSITORY")
	api := env("CAA_API_URL", "https://api.github.com")

	raw, err := os.ReadFile(must("CAA_EVENT_PATH"))
	if err != nil {
		fail("read event: %v", err)
	}
	var event payload
	if err := json.Unmarshal(raw, &event); err != nil {
		fail("parse event: %v", err)
	}

	cfg := Config{
		Sentence:  must("CAA_SENTENCE"),
		Allowlist: splitList(os.Getenv("CAA_ALLOWLIST")),
	}

	caller := newClient(api, must("CAA_GITHUB_TOKEN"))

	ev := Event{Name: eventName, RepoID: event.Repository.ID}
	switch eventName {
	case "pull_request_target":
		if event.PullRequest == nil {
			fail("pull_request_target event carries no pull request")
		}
		ev.OnPullRequest = true
		ev.PullRequestNo = event.PullRequest.Number
		ev.Author = event.PullRequest.User.Login
		ev.HeadSHA = event.PullRequest.Head.SHA
	case "issue_comment":
		if event.Issue == nil || event.Comment == nil {
			fail("issue_comment event is missing its issue or comment")
		}
		// A plain issue has no pull request. The old automation died here with
		// a GraphQL error; this exits quietly instead.
		if len(event.Issue.PullRequest) == 0 || string(event.Issue.PullRequest) == "null" {
			fmt.Println("comment is on an issue, not a pull request: nothing to do")
			return
		}
		ev.OnPullRequest = true
		ev.PullRequestNo = event.Issue.Number
		ev.CommentBody = event.Comment.Body
		ev.CommentAuthor = event.Comment.User.Login
		ev.CommentAuthorID = event.Comment.User.ID
		ev.CommentID = event.Comment.ID
		ev.CommentCreatedAt = event.Comment.CreatedAt

		pr, err := caller.pullRequest(repo, ev.PullRequestNo)
		if err != nil {
			fail("read pull request %d: %v", ev.PullRequestNo, err)
		}
		ev.Author = pr.User.Login
		ev.HeadSHA = pr.Head.SHA
	default:
		fail("unsupported event %q", eventName)
	}

	ledgerRepo := env("CAA_LEDGER_REPOSITORY", "mas-bandwidth/.github")
	ledgerPath := env("CAA_LEDGER_PATH", "signatures/caa.json")
	ledgerBranch := env("CAA_LEDGER_BRANCH", "cla-signatures")
	ledgerClient := newClient(api, must("CAA_LEDGER_TOKEN"))

	ledgerRaw, blobSHA, err := ledgerClient.readLedger(ledgerRepo, ledgerPath, ledgerBranch)
	if err != nil {
		fail("read ledger: %v", err)
	}
	ledger, err := ParseLedger(ledgerRaw)
	if err != nil {
		fail("%v", err)
	}

	decision := Decide(ev, ledger, cfg)
	if decision.Skip {
		fmt.Println("nothing to do")
		return
	}

	if decision.Sign {
		if err := sign(ledgerClient, ledgerRepo, ledgerPath, ledgerBranch, blobSHA, ledger, decision.Signature, repo); err != nil {
			fail("%v", err)
		}
		fmt.Printf("recorded a signature from %s on %s#%d\n", decision.Signature.Name, repo, ev.PullRequestNo)
	}

	if decision.Comment {
		posted, err := caller.hasComment(repo, ev.PullRequestNo, marker)
		if err != nil {
			fail("read comments on %s#%d: %v", repo, ev.PullRequestNo, err)
		}
		if !posted {
			if err := caller.postComment(repo, ev.PullRequestNo, askToSign(cfg.Sentence)); err != nil {
				fail("post the ask to sign on %s#%d: %v", repo, ev.PullRequestNo, err)
			}
			fmt.Printf("asked %s to sign on %s#%d\n", ev.Author, repo, ev.PullRequestNo)
		}
	}

	description := "Waiting for the Contributor Assignment Agreement signature"
	if decision.Status == "success" {
		description = "Contributor Assignment Agreement signed"
	}
	runURL := fmt.Sprintf("%s/%s/actions/runs/%s",
		env("CAA_SERVER_URL", "https://github.com"), repo, os.Getenv("CAA_RUN_ID"))
	if err := caller.setStatus(repo, ev.HeadSHA, decision.Status, env("CAA_STATUS_CONTEXT", "caa"), description, runURL); err != nil {
		fail("set the commit status on %s: %v", ev.HeadSHA, err)
	}
	fmt.Printf("status %s on %s\n", decision.Status, ev.HeadSHA)
}

// sign appends one signature and writes the ledger back. A write that loses a
// race with another run is retried against the ledger as it now stands, and
// the append is a no-op if that other run already recorded the same person.
func sign(c *client, repo, path, branch, blobSHA string, ledger *Ledger, sig Signature, from string) error {
	message := fmt.Sprintf("Record the CAA signature of %s on %s#%d", sig.Name, from, sig.PullRequestNo)

	for attempt := 0; ; attempt++ {
		if !ledger.Append(sig) {
			return nil
		}
		raw, err := ledger.Marshal()
		if err != nil {
			return err
		}
		status, err := c.writeLedger(repo, path, branch, blobSHA, message, raw)
		if err == nil {
			return nil
		}
		if attempt == 4 || (status != http.StatusConflict && status != http.StatusUnprocessableEntity) {
			return fmt.Errorf("write ledger: %w", err)
		}

		current, currentSHA, err := c.readLedger(repo, path, branch)
		if err != nil {
			return fmt.Errorf("reread ledger: %w", err)
		}
		if ledger, err = ParseLedger(current); err != nil {
			return err
		}
		blobSHA = currentSHA
	}
}

func askToSign(sentence string) string {
	document := env("CAA_DOCUMENT_URL", "https://github.com/mas-bandwidth/.github/blob/main/CAA.md")
	return strings.Join([]string{
		marker,
		"Thanks for the contribution. Before it can be merged, please read the",
		"[Contributor Assignment Agreement](" + document + ") and sign it by posting",
		"the sentence below, on its own, as a comment on this pull request.",
		"",
		"> " + sentence,
	}, "\n")
}
