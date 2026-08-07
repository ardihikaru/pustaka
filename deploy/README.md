# Pustaka deployment

`pustaka.mfardiansyah.id` and `dev.pustaka.mfardiansyah.id` are separate
root-domain stacks. `clino` terminates HTTPS and forwards plain HTTP through
WireGuard to `rnd`; Traefik is not exposed on any LAN interface.

| Environment | Hostname | rnd tunnel port | Internal network |
|---|---|---:|---|
| Production | `pustaka.mfardiansyah.id` | `10.10.0.2:18401` | `172.30.14.0/24` |
| Development | `dev.pustaka.mfardiansyah.id` | `10.10.0.2:19401` | `172.30.15.0/24` |

## rnd

From the checkout, preserve a dirty host checkout before updating it, then
create one secret file per environment. Do not reuse session secrets.

```bash
cd /path/to/pustaka
git stash push --include-untracked -m 'pre-pustaka-deploy'
git pull --ff-only

cp deploy/stacks/docs.env.example deploy/stacks/dev.docs.env
cp deploy/stacks/docs.env.example deploy/stacks/prod.docs.env
# Edit both files: set PUSTAKA_AUTH_SECRET to different `openssl rand -hex 32` values.
chmod 600 deploy/stacks/dev.docs.env deploy/stacks/prod.docs.env

docker compose -f deploy/stacks/compose.dev.yaml build
docker compose -f deploy/stacks/compose.dev.yaml up -d
docker compose -f deploy/stacks/compose.prod.yaml build
docker compose -f deploy/stacks/compose.prod.yaml up -d
```

Check the rendered manifests before a rollout:

```bash
docker compose -f deploy/stacks/compose.dev.yaml config
docker compose -f deploy/stacks/compose.prod.yaml config
```

## clino

Install the HTTP-only bootstrap file first: an HTTPS configuration cannot pass
`nginx -t` before its certificate exists. After HTTP-01 issuance replace it
with the full edge configuration, which maps the two hostnames to their
WireGuard ports and preserves Host, client IP, and the external HTTPS scheme.

```bash
sudo install -d -m 755 /var/www/acme
sudo install -m 644 deploy/edge/pustaka.bootstrap.nginx.conf /etc/nginx/conf.d/pustaka.conf
sudo nginx -t && sudo systemctl reload nginx

# Reuse the account/contact already configured for the host. Keep these
# certificate lineages separate, so a development DNS problem cannot affect
# production renewal.
sudo certbot certonly --webroot -w /var/www/acme \
  -d pustaka.mfardiansyah.id --cert-name pustaka \
  --deploy-hook 'nginx -t && systemctl reload nginx'
sudo certbot certonly --webroot -w /var/www/acme \
  -d dev.pustaka.mfardiansyah.id --cert-name pustaka-dev \
  --deploy-hook 'nginx -t && systemctl reload nginx'
sudo install -m 644 deploy/edge/pustaka.nginx.conf /etc/nginx/conf.d/pustaka.conf
sudo nginx -t && sudo systemctl reload nginx
```

Verify tunnel routing from `clino` and the public HTTPS endpoints after rollout:

```bash
curl -i -H 'Host: pustaka.mfardiansyah.id' http://10.10.0.2:18401/__pustaka/info
curl -i -H 'Host: dev.pustaka.mfardiansyah.id' http://10.10.0.2:19401/__pustaka/info
curl -i https://pustaka.mfardiansyah.id/
curl -i https://dev.pustaka.mfardiansyah.id/
```
