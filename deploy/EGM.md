# deploy — reference edge + container topology

## Overview

Optional infrastructure templates for running Pustaka behind a TLS edge, with
independent dev and prod stacks. The human guide is `deploy/README.md`; this
file is the contract. Topology:

```
browser --HTTPS--> edge host: nginx (TLS, edge/pustaka.nginx.conf)
                     | plain HTTP over a private tunnel (WireGuard)
                     v
                   app host: Traefik per stack (traefik/dynamic.yml)
                     --> pustaka:8080 (image from /Dockerfile)
```

The single-host alternative is the root `docker-compose.yml` with `.env.example`.

## Interface

- `stacks/compose.dev.yaml`, `stacks/compose.prod.yaml` — Compose projects
  `pustaka-dev` / `pustaka-prod`, each with services `traefik` (`traefik:v3.3`)
  and `pustaka` (built from `../..` with the root `Dockerfile`), on a private
  bridge network `pustaka-{dev,prod}-int`. Run as
  `docker compose --env-file deploy/stacks/<env>.compose.env -f deploy/stacks/compose.<env>.yaml up -d --build`.
- `stacks/compose.env.example` — required Compose variables: `PUSTAKA_BIND`
  (Traefik publish address), `PUSTAKA_SUBNET` (bridge subnet), `PUSTAKA_EDGE_IP`
  (only trusted source of `X-Forwarded-*`).
- `stacks/docs.env.example` — runtime env for the `pustaka` service:
  `PUSTAKA_AUTH`, `PUSTAKA_AUTH_USER`, `PUSTAKA_AUTH_PASS`,
  `PUSTAKA_AUTH_SECRET`, `PUSTAKA_AUTH_TTL`, `PUSTAKA_AUTH_SECURE`,
  `PUSTAKA_AUTH_PUBLIC_HOME` (parsed by `[main.go:loadAuthConfig]`).
  `PUSTAKA_AUTH_TRUSTED_PROXIES` is not set here: the compose file derives it
  as `${PUSTAKA_SUBNET},${PUSTAKA_EDGE_IP}`.
- Traefik has no static config file: its entry point, trusted edge IP and file
  provider are `command:` flags in the compose files.
- `traefik/dynamic.yml` — one router `pustaka`, `PathPrefix(`/`)` on entrypoint
  `web`, to `http://pustaka:8080`.
- `edge/pustaka.nginx.conf` — `map $host $pustaka_backend` (prod
  `10.0.0.2:18401`, dev `10.0.0.2:19401`), HTTP→HTTPS redirect, ACME webroot
  `/var/www/acme`, separate Let's Encrypt lineages `pustaka` and `pustaka-dev`.
- `edge/pustaka.bootstrap.nginx.conf` — HTTP-only config for the first
  HTTP-01 issuance; replaced by `pustaka.nginx.conf` afterwards.

## Invariants

- Pustaka owns the origin root: no path prefix / `stripPrefix`, because
  `[main.go:authRoutePrefix]` redirects and the login template's `{{BASE}}` are
  root-absolute (`/`).
- TLS terminates at the edge and the container sees plain HTTP, so
  `PUSTAKA_AUTH_SECURE=1` must be explicit; the default `auto` in
  `[main.go:auth.secure]` keys off `r.TLS` and would issue non-Secure cookies.
- `PUSTAKA_AUTH_SECRET` must be set and differ per environment; when unset,
  `[main.go:loadAuthConfig]` generates an ephemeral key and every restart signs
  everyone out.
- `${VAR:?…}` interpolation makes Compose refuse to render with a missing
  network value; dev and prod need distinct `PUSTAKA_BIND` ports and
  non-overlapping `PUSTAKA_SUBNET`s.
- `PUSTAKA_BIND` must be the private tunnel address, never a LAN/public one.
- Login throttling (`[main.go:authMaxAttempts]` = 5 failures,
  `[main.go:authLockout]` = 1 min) is keyed by `[main.go:clientIP]`. Because
  the stack trusts its own subnet and the edge IP, the key is the real visitor
  address from `X-Forwarded-For`. If the edge connection were source-NATed,
  every visitor would share one bucket again.
- Runtime env files (`stacks/dev.docs.env`, `stacks/prod.docs.env`,
  `stacks/*.compose.env`) are git-ignored; only `*.example` templates are
  committed.

## Side Effects

- Builds a container image and starts long-running `traefik` and `pustaka`
  containers; creates the named Docker networks.
- Traefik binds `PUSTAKA_BIND` on the app host; nginx binds 80/443 on the edge
  and reads certificates from `/etc/letsencrypt/live/`.
- Traefik writes an access log (`--accesslog=true`).

## Verification

- Engine: `go test ./... && go build -o pustaka . && ./pustaka check ./docs`
- Manifests: `docker compose --env-file deploy/stacks/compose.env.example -f deploy/stacks/compose.dev.yaml config`
  (needs a `stacks/dev.docs.env` to exist), and `nginx -t` on the edge host.
