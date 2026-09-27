# 07 — Research: Traefik forwarding to a host service with Let's Encrypt

**Type:** research
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

How does the gateway Traefik (deployed by the runner as a container, docker provider, `exposedByDefault=false`) also (a) forward one domain to the Nexul server running on the host at `host.docker.internal:<port>` and (b) obtain and renew a Let's Encrypt certificate for it and for later exposures, using only what a run-strategy deploy can express (image, command, ports, mounts, env)? Compare: Traefik v3 labels on its own container (can a service point at a URL?), the file provider with a mounted dynamic config the runner writes, and ACME HTTP-01 storage (volume for `acme.json`). Answer from Traefik's official docs (Context7 or the docs site), with the exact flags.

## Answer

Sources: Traefik v3 docs, [boot environment](https://doc.traefik.io/traefik/reference/install-configuration/boot-environment/),
[configuration options](https://doc.traefik.io/traefik/reference/install-configuration/configuration-options/),
[ACME](https://doc.traefik.io/traefik/https/acme/), [file provider](https://doc.traefik.io/traefik/providers/file/),
[services](https://doc.traefik.io/traefik/routing/services/), and the image's
[entrypoint](https://github.com/traefik/traefik-library-image).

**Routing to the host.** Static config (entry points, providers, resolvers) can come from env or flags, never
both, and neither can declare a router. Routes come only from a provider:

- Docker labels can point a service at a URL (`traefik.http.services.<s>.loadBalancer.servers[0].url=...`), but
  the runner cannot set labels (`runArgs` has none), and adding them means a new stack column, a protocol field,
  and `traefik.enable=true` on Traefik itself.
- The file provider works with what a run deploy already expresses. Traefik reads
  `TRAEFIK_PROVIDERS_FILE_FILENAME=/etc/traefik/nexul.yml`. The container's command is
  `sh`, `-c`, `mkdir -p /etc/traefik && printf '%s' "$NEXUL_TRAEFIK_DYNAMIC" > /etc/traefik/nexul.yml && exec traefik`
  (the image's entrypoint runs a command that is not a Traefik subcommand through the shell), and the route
  arrives in the `NEXUL_TRAEFIK_DYNAMIC` env var as JSON, which the YAML loader reads unchanged. Nothing has to
  exist on the host, and the stack's env carries the domain, so a redeploy with a new domain updates the route.

Chosen: the file provider. The route is
`{"http":{"routers":{"nexul":{"rule":"Host(`<domain>`)","entryPoints":["websecure"],"service":"nexul","tls":{"certResolver":"letsencrypt"}}},"services":{"nexul":{"loadBalancer":{"servers":[{"url":"http://host.docker.internal:<port>"}]}}}}}`.
The docker provider stays on (`exposedByDefault=false`) for exposures.

**Let's Encrypt over HTTP-01.** HTTP-01 is compatible with the http to https redirect and needs port 80
reachable:

```
TRAEFIK_ENTRYPOINTS_WEB_ADDRESS=:80
TRAEFIK_ENTRYPOINTS_WEB_HTTP_REDIRECTIONS_ENTRYPOINT_TO=websecure
TRAEFIK_ENTRYPOINTS_WEB_HTTP_REDIRECTIONS_ENTRYPOINT_SCHEME=https
TRAEFIK_ENTRYPOINTS_WEBSECURE_ADDRESS=:443
TRAEFIK_ENTRYPOINTS_WEBSECURE_HTTP_TLS_CERTRESOLVER=letsencrypt
TRAEFIK_CERTIFICATESRESOLVERS_LETSENCRYPT_ACME_STORAGE=/letsencrypt/acme.json
TRAEFIK_CERTIFICATESRESOLVERS_LETSENCRYPT_ACME_HTTPCHALLENGE_ENTRYPOINT=web
TRAEFIK_CERTIFICATESRESOLVERS_LETSENCRYPT_ACME_EMAIL=<email>   # optional
```

`acme.json` lives in the named volume `nexul-traefik-acme` mounted at `/letsencrypt`, so redeploys keep the
account and certificates, and Traefik renews them itself. The entry-point default resolver gives every router on
`websecure` a certificate, exposures included once they carry routers. Swapping in the staging CA is
`TRAEFIK_CERTIFICATESRESOLVERS_LETSENCRYPT_ACME_CASERVER=https://acme-staging-v02.api.letsencrypt.org/directory`.
