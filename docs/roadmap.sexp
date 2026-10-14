; The .github roadmap: the data ROADMAP.md is generated from.
; Same shape as nova-sprint's docs/roadmap.sexp (internal/roadmapdoc's package comment):
; data, never evaluated. Edit this file, then regenerate ROADMAP.md with nova-sprint's
; generator, from a nova-sprint checkout:
;   go build -o /tmp/roadmap ./tools/roadmap && (cd <this repo> && /tmp/roadmap --file docs/roadmap.sexp --out ROADMAP.md)
; Never edit ROADMAP.md by hand. Created 2026-10-14 when this repository's open GitHub issues and
; unfinished pull requests were moved into it.
(roadmap "v1"
 :title ".github roadmap"
 :text "Planned work for .github. The items below were open GitHub issues and pull requests, moved here on 2026-10-14 when roadmaps became the standard for future work (Glenn, 2026-10-14). Each item names its issue or pull request; the branches of moved pull requests are kept to resume from. Landed work is in git."

 :groups
 (
  (group "github-issues" :title "Moved from GitHub issues on 2026-10-14"
   :text "Open GitHub issues of mas-bandwidth/.github, recorded here on 2026-10-14 when roadmaps became the standard for future work (Glenn, 2026-10-14). Each item names its issue; the issue is closed with a pointer here.")
  (group "github-prs" :title "Unfinished pull requests, recorded on 2026-10-14"
   :text "Open pull requests of mas-bandwidth/.github whose work is not finished, recorded here on 2026-10-14 and closed unmerged. Each names its branch and head commit; the branch is kept to resume from.")
 )

 :items
 (
  (item "issue-3" :group "github-issues" :area ".github"
   :title "CI over the two-minute law: eleven per-commit workflows, ranked, with the fix per repo (2026-09-13 survey)"
   :text "Glenn's law, 2026-09-13: CI that runs on every commit or PR finishes in one to two minutes, or it is fixed, split, or moved to a tier that does not run all the time. Survey by a read-only child over 73 repos, 40 with workflows, 39 in-scope workflows with runs in the last 14 days (median of the last 5 runs; slowest job of the latest run)."
   :date "2026-10-14"
   :origin "mas-bandwidth/.github issue #3, opened 2026-09-13 by rowan-claude; moved here 2026-10-14"
   :kept (:github-issue "3" :labels ""))
  (item "pr-2" :group "github-prs" :area ".github"
   :title "Own the CAA gate: a reusable workflow and a Go program in this repository"
   :text "Replaces the archived third-party CAA action with a reusable workflow and a Go program this organization owns. The gate ran on contributor-assistant/github-action@v2.6.1. That repository is archived, v2.6.1 (2024-09-26) is its last release, and it holds a personal access token with write on every library repository."
   :date "2026-10-14"
   :origin "mas-bandwidth/.github pull request #2, branch caa-reusable-workflow@6a7011c8 onto main, opened 2026-09-04 by gafferongames; closed unmerged 2026-10-14, the branch is kept to resume from"
   :kept (:github-pr "2" :branch "caa-reusable-workflow" :head "6a7011c8e9f5952ce16a5e261336b7ac67736379"))
 ))
