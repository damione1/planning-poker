# Deployment Guide (Lightsail)

Production is a Lightsail instance (`planning-poker`, us-east-1) running Docker Compose: Traefik + `ghcr.io/damione1/planning-poker`.

```
git tag vX.Y.Z && git push origin vX.Y.Z
  → tests
  → GitHub Release
  → build/push GHCR (amd64 + arm64)
  → mint a 60s Lightsail SSH cert
  → /opt/planning-poker/scripts/deploy.sh
  → GET https://pokerplanning.net/monitoring/health
```

## GitHub config

**Secrets:** `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` (IAM user `github-actions-planning-poker`)

**Variables:**

| Name | Value |
|---|---|
| `AWS_REGION` | `us-east-1` |
| `DOMAIN_NAME` | `pokerplanning.net` |
| `LIGHTSAIL_INSTANCE_NAME` | `planning-poker` |
| `LETS_ENCRYPT_EMAIL` | Let's Encrypt contact |

IAM policy: `terraform/iam-policy.json` (`lightsail:GetInstanceAccessDetails` + `GetInstance`).

## Manual deploy

```bash
export AWS_PROFILE=Damien AWS_REGION=us-east-1 LIGHTSAIL_INSTANCE_NAME=planning-poker
./scripts/ci-deploy-lightsail.sh 1.2.3 1.2.3
```

On-box equivalent: `ssh … '/opt/planning-poker/scripts/deploy.sh 1.2.3 1.2.3'`

The remote script pulls the GHCR tag, retags `latest`, and recreates **only** the app container (Traefik stays up so the cert is not bounced).

## Bootstrap (new instance)

`terraform/lightsail-user-data.sh` — Docker, compose, Traefik, first `up`. Data lives on the bundle disk at `/mnt/data/pb_data`.
