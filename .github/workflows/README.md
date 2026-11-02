# GitHub Actions workflows

## `deploy.yml`

Release tags `v*.*.*` run tests, publish a GitHub release and multi-platform GHCR image, then deploy through GitHub OIDC and the restricted Systems Manager document. The target is Applications account `675704851310`.

Variables: `AWS_REGION`, `AWS_DEPLOY_ROLE_ARN`, `SSM_INSTANCE_ID`, `DOMAIN_NAME`. AWS access keys and SSH private keys are not required.

```sh
git tag v1.2.3
git push origin v1.2.3
gh run watch
```

## `verify-aws-access.yml`

Manual dispatch verifies OIDC and local HTTPS on the managed node. Supplying an existing `redeploy_version` also redeploys that image version.

## `test.yml`

Runs on push/PR to `master` and is reused by the release workflow.

See [the AWS deployment guide](../../terraform/DEPLOY.md) for Terraform ownership, profiles, backups and recovery.
