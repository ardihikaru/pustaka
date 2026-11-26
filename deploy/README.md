# Pustaka deployment

A reference topology for running Pustaka behind a TLS edge, with separate
development and production stacks. Nothing here is required — a single
`pustaka serve ./docs --prod` behind any reverse proxy works too (see
`docs/guide/deploy/production.html`). Use this when you want one host to
terminate HTTPS and another to run the containers.

```
browser ──HTTPS──▶ edge host (Nginx, TLS)
                     │  plain HTTP over a private tunnel (e.g. WireGuard)
                     ▼
                   app host ── Traefik (per stack) ──▶ pustaka:8080
```

| Environment | Example hostname | App-host bind | Stack subnet |
|---|---|---:|---|
| Production | `docs.example.com` | `10.0.0.2:18401` | `172.31.10.0/24` |
| Development | `dev.docs.example.com` | `10.0.0.2:19401` | `172.31.11.0/24` |

All addresses above are placeholders. The real values live in untracked env
files on the app host, so the committed manifests are identical everywhere.

| File | Tracked | Holds |
|---|---|---|
| `stacks/compose.{dev,prod}.yaml` | yes | The stack: Traefik + Pustaka |
| `stacks/compose.env.example` | yes | Template for the network values |
| `stacks/{dev,prod}.compose.env` | **no** | `PUSTAKA_BIND`, `PUSTAKA_SUBNET`, `PUSTAKA_EDGE_IP` |
| `stacks/docs.env.example` | yes | Template for the login settings |
| `stacks/{dev,prod}.docs.env` | **no** | `PUSTAKA_AUTH_*`, including the session secret |
| `traefik/dynamic.yml` | yes | Routes the origin root to `pustaka:8080` |
| `edge/*.nginx.conf` | yes | Edge templates — replace the example hostnames and IPs |

## App host

### First install

Create the four runtime files once. `cp -n` never overwrites an existing
file. Do not reuse session secrets across environments.

```bash
cd /path/to/pustaka
for env in dev prod; do
  cp -n deploy/stacks/compose.env.example deploy/stacks/$env.compose.env
  cp -n deploy/stacks/docs.env.example    deploy/stacks/$env.docs.env
done
# Fill in every value. Give dev and prod different PUSTAKA_BIND ports and
# subnets, and different PUSTAKA_AUTH_SECRET values (openssl rand -hex 32).
chmod 600 deploy/stacks/*.compose.env deploy/stacks/*.docs.env
```

The compose file derives `PUSTAKA_AUTH_TRUSTED_PROXIES` from
`PUSTAKA_SUBNET` and `PUSTAKA_EDGE_IP`, so the login throttle sees each
visitor's real address instead of Traefik's — no second copy to keep in sync.
Traefik's static configuration is the `command:` flags in the compose file
(entry point, trusted edge IP, file provider for `traefik/dynamic.yml`); there
is no `traefik.yml`.

### Roll out (first install and every update)

```bash
cd /path/to/pustaka
git stash push --include-untracked -m 'pre-pustaka-deploy'   # only if tracked files are dirty
git pull --ff-only
for env in dev prod; do
  docker compose --env-file deploy/stacks/$env.compose.env \
    -f deploy/stacks/compose.$env.yaml config >/dev/null &&
  docker compose --env-file deploy/stacks/$env.compose.env \
    -f deploy/stacks/compose.$env.yaml up -d --build
done
```

An empty or missing value fails the render with `required variable … is
missing`, rather than publishing on a guessed address. The runtime env files
are git-ignored, so `git stash` leaves them in place.

### Upgrading from the hard-coded layout (≤ v0.8)

Earlier manifests hard-coded the bind address, subnet and edge IP and used
`deploy/stacks/traefik.yml`. Before pulling, save the current render so you
can copy the old values and compare:

```bash
for env in dev prod; do
  docker compose -f deploy/stacks/compose.$env.yaml config > /tmp/pustaka-old.$env.yaml
done
git pull --ff-only
# Create $env.compose.env with the ports/subnet/edge IP from the saved
# renders (keep the existing $env.docs.env untouched), then:
for env in dev prod; do
  docker compose --env-file deploy/stacks/$env.compose.env \
    -f deploy/stacks/compose.$env.yaml config | diff /tmp/pustaka-old.$env.yaml - | grep -E 'published|host_ip|subnet'
done
```

No `published`/`host_ip`/`subnet` lines in the diff means the networks are
recreated identically. Run `up -d --build` right after the pull: the old
Traefik container bind-mounts the deleted `traefik.yml`, and if it restarts
first Docker may create a directory at that path (remove it if so).

## Edge host

Install the HTTP-only bootstrap file first: an HTTPS configuration cannot pass
`nginx -t` before its certificate exists. After HTTP-01 issuance replace it
with the full configuration, which maps each hostname to its tunnel port and
preserves Host, client IP, and the external HTTPS scheme. Replace the example
hostnames and addresses in both files before installing them.

```bash
sudo install -d -m 755 /var/www/acme
sudo install -m 644 deploy/edge/pustaka.bootstrap.nginx.conf /etc/nginx/conf.d/pustaka.conf
sudo nginx -t && sudo systemctl reload nginx

# Keep separate certificate lineages so a development DNS problem cannot
# affect production renewal.
sudo certbot certonly --webroot -w /var/www/acme \
  -d docs.example.com --cert-name pustaka \
  --deploy-hook 'nginx -t && systemctl reload nginx'
sudo certbot certonly --webroot -w /var/www/acme \
  -d dev.docs.example.com --cert-name pustaka-dev \
  --deploy-hook 'nginx -t && systemctl reload nginx'
sudo install -m 644 deploy/edge/pustaka.nginx.conf /etc/nginx/conf.d/pustaka.conf
sudo nginx -t && sudo systemctl reload nginx
```

## Verify

From the edge host, then publicly:

```bash
curl -i -H 'Host: docs.example.com'     http://10.0.0.2:18401/__pustaka/info
curl -i -H 'Host: dev.docs.example.com' http://10.0.0.2:19401/__pustaka/info
curl -i https://docs.example.com/
curl -i https://dev.docs.example.com/
```

`/__pustaka/info` reports the running `version`; confirm it matches the
release you just pulled. To confirm the throttle is per visitor, fail five
logins from one client and then sign in from another: the second must not be
locked out. If it is, the edge connection is being source-NATed and Traefik
never sees the edge IP.
