package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// client talks to one GitHub API host with one token. The program builds two:
// one on the calling repository's GITHUB_TOKEN for pull request metadata,
// comments and the commit status, and one on CAA_LEDGER_TOKEN for the ledger.
// Nothing shares a token across those two jobs.
type client struct {
	api   string
	token string
	http  *http.Client
}

func newClient(api, token string) *client {
	return &client{
		api:   strings.TrimSuffix(api, "/"),
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *client) do(method, path string, body any, out any) (int, error) {
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		payload = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.api+path, payload)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}
	if resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(raw)))
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("%s %s: decode response: %w", method, path, err)
		}
	}
	return resp.StatusCode, nil
}

type pullRequest struct {
	Number int `json:"number"`
	User   struct {
		Login string `json:"login"`
	} `json:"user"`
	Head struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

func (c *client) pullRequest(repo string, number int) (*pullRequest, error) {
	pr := &pullRequest{}
	_, err := c.do(http.MethodGet, fmt.Sprintf("/repos/%s/pulls/%d", repo, number), nil, pr)
	return pr, err
}

type issueComment struct {
	Body string `json:"body"`
}

// hasComment reports whether a comment carrying marker is already on the
// thread, so the ask to sign is posted once and not once per push.
func (c *client) hasComment(repo string, number int, marker string) (bool, error) {
	for page := 1; page <= 10; page++ {
		var comments []issueComment
		path := fmt.Sprintf("/repos/%s/issues/%d/comments?per_page=100&page=%d", repo, number, page)
		if _, err := c.do(http.MethodGet, path, nil, &comments); err != nil {
			return false, err
		}
		for _, comment := range comments {
			if strings.Contains(comment.Body, marker) {
				return true, nil
			}
		}
		if len(comments) < 100 {
			return false, nil
		}
	}
	return false, nil
}

func (c *client) postComment(repo string, number int, body string) error {
	_, err := c.do(http.MethodPost, fmt.Sprintf("/repos/%s/issues/%d/comments", repo, number),
		map[string]string{"body": body}, nil)
	return err
}

func (c *client) setStatus(repo, sha, state, context, description, targetURL string) error {
	_, err := c.do(http.MethodPost, fmt.Sprintf("/repos/%s/statuses/%s", repo, sha), map[string]string{
		"state":       state,
		"context":     context,
		"description": description,
		"target_url":  targetURL,
	}, nil)
	return err
}

type contentsFile struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	SHA      string `json:"sha"`
}

// readLedger returns the ledger bytes and the blob sha the write has to send
// back. A ledger file that does not exist yet reads as empty with an empty sha.
func (c *client) readLedger(repo, path, branch string) ([]byte, string, error) {
	file := &contentsFile{}
	endpoint := fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, path, url.QueryEscape(branch))
	status, err := c.do(http.MethodGet, endpoint, nil, file)
	if status == http.StatusNotFound {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	if file.Encoding != "base64" {
		return nil, "", fmt.Errorf("read ledger: unexpected encoding %q", file.Encoding)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(file.Content, "\n", ""))
	if err != nil {
		return nil, "", fmt.Errorf("read ledger: %w", err)
	}
	return raw, file.SHA, nil
}

func (c *client) writeLedger(repo, path, branch, blobSHA, message string, raw []byte) (int, error) {
	body := map[string]string{
		"message": message,
		"content": base64.StdEncoding.EncodeToString(raw),
		"branch":  branch,
	}
	if blobSHA != "" {
		body["sha"] = blobSHA
	}
	return c.do(http.MethodPut, fmt.Sprintf("/repos/%s/contents/%s", repo, path), body, nil)
}
