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

From the checkout, preserve any host-local changes before updating, then create
the four runtime files once. Do not reuse session secrets across environments.

```bash
cd /path/to/pustaka
git stash push --include-untracked -m 'pre-pustaka-deploy'   # only if dirty
git pull --ff-only

for env in dev prod; do
  cp deploy/stacks/compose.env.example deploy/stacks/$env.compose.env
  cp deploy/stacks/docs.env.example    deploy/stacks/$env.docs.env
done
# Edit all four. Give dev and prod different PUSTAKA_BIND ports and subnets,
# and different PUSTAKA_AUTH_SECRET values (openssl rand -hex 32).
chmod 600 deploy/stacks/*.compose.env deploy/stacks/*.docs.env
```

Render, then roll out each stack:

```bash
for env in dev prod; do
  docker compose --env-file deploy/stacks/$env.compose.env \
    -f deploy/stacks/compose.$env.yaml config >/dev/null &&
  docker compose --env-file deploy/stacks/$env.compose.env \
    -f deploy/stacks/compose.$env.yaml up -d --build
done
```

A missing value fails the render with `required variable … is missing`, rather
than publishing on a default address.

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
release you just pulled.
