# .github

Org-wide GitHub configuration for Más Bandwidth LLC.

- [`CAA.md`](CAA.md) is the Contributor Assignment Agreement. Every outside
  contributor signs it once, and the signature counts in every repository.
- [`.github/workflows/caa.yml`](.github/workflows/caa.yml) is the reusable
  workflow that holds a pull request until its author has signed. Each library
  repository calls it from its own `cla.yml`.
- [`tools/caa`](tools/caa) is the program behind that workflow. `go test ./...`
  runs its tests.
- `FUNDING.yml` puts the Sponsor button on every repository.

The signature ledger is `signatures/caa.json` on the `cla-signatures` branch of
this repository. Its entries look like this:

```json
{
  "name": "<github login>",
  "id": 12345,
  "comment_id": 67890,
  "created_at": "2026-08-23T11:04:08Z",
  "repoId": 59925747,
  "pullRequestNo": 331
}
```

To record a signature that arrived some other way, such as on an issue rather
than a pull request, append an entry by hand on that branch and commit it with
a message naming the signer and the thread. Any later comment on the pull
request re-runs the gate and turns the status green.
