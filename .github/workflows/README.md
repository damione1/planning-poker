# GitHub Actions Workflows

## `deploy.yml` — production

Trigger: tag `v*.*.*`

```
tests → GitHub Release → GHCR (amd64/arm64) → Lightsail SSH deploy → health check
```

See [terraform/DEPLOY.md](../../terraform/DEPLOY.md).

**Secrets:** `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`

**Variables:** `AWS_REGION`, `DOMAIN_NAME`, `LIGHTSAIL_INSTANCE_NAME`, `LETS_ENCRYPT_EMAIL`

```bash
git tag v1.2.3
git push origin v1.2.3
gh run watch
```

## `test.yml` — CI

Runs on push/PR to `master`, and is reused by the deploy workflow.
