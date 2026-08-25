# demo-api

A small HTTP service whose job is to be deployed.

It exists so the delivery pipeline has something real to carry:

```
push here → Jenkins builds and tests → ghcr.io/camircode/demo-api@sha256:…
          → Jenkins commits that digest to camircode/gitops
          → Argo CD rolls it out → https://api.camir.tech
```

Nothing in this repository deploys anything. Jenkins changes one line in the
GitOps repository and Argo CD does the rest, which is why that repository's git
log is the real deployment history.

## Endpoints

| Path | Purpose |
|---|---|
| `GET /health` | Liveness and readiness. Reports the version and commit that answered. |
| `GET /` | The same metadata, plus the pod's hostname. |

## Locally

```bash
go test ./...
go run ./cmd/api
```
