# Planning Poker AWS deployment

Production runs in **Applications account `675704851310`**, on one Lightsail `micro_3_0` instance in `us-east-1a`. `https://pokerplanning.net` resolves to static IPv4 `184.196.207.251`. Only TCP 80 and 443 are public; Systems Manager (SSM) handles deployment and administration.

The former instance and static IP in management account `912928333407` were deleted after the November 2, 2026 migration. There is one application server, with the same $7 USD/month base bundle as before. Snapshots, backups and security services have separate usage charges. See [Lightsail pricing](https://aws.amazon.com/lightsail/pricing/).

## Terraform ownership

| Directory | Resources | Remote state key |
|---|---|---|
| `terraform/applications/` | Production instance, IP, firewall, snapshots, app IAM/OIDC, SSM documents and daily backup | `planning-poker-applications/terraform.tfstate` |
| `terraform/dns/` | Existing Cloudflare A record for `pokerplanning.net` | `planning-poker-dns/terraform.tfstate` |
| Sibling `homelab/terraform/aws-account/` | Organization, SSO, CloudTrail, central storage, alerts and management-account controls | `account-security/terraform.tfstate` |
| Sibling `homelab/terraform/aws-applications/` | Applications security baseline in 17 enabled regions | `applications-account/terraform.tfstate` |
| `terraform/` | Retired management-account application state; no managed resources | `production/terraform.tfstate`, original state bucket |

Active states use private, encrypted, versioned bucket `panetone-terraform-state-912928333407`, with DynamoDB lock table `panetone-terraform-locks`. The old empty application state remains in `planning-poker-terraform-state-72cf7d8d` for history. Never commit state, plans or credentials. Provider lock files are tracked.

```sh
aws sso login --profile aws-admin
aws sso login --profile aws-applications-admin
terraform -chdir=terraform/applications init
terraform -chdir=terraform/applications plan
# Review the plan before applying it.
terraform -chdir=terraform/applications apply
terraform/dns/tf.sh init
terraform/dns/tf.sh plan
terraform/dns/tf.sh apply
```

`aws-admin` administers management account `912928333407`; `aws-applications-admin` administers `675704851310`. Backend profile `aws-terraform` uses `aws configure export-credentials --profile aws-admin --format process` as a `credential_process`, supporting Terraform 1.5 with temporary SSO credentials and no static keys. Read-only audit profiles are `aws-audit` and `aws-applications-audit`.

The DNS wrapper reads Keychain service `cloudflare-dns-tf-token`. Its token needs Zone Read and DNS Edit on **pokerplanning.net only**. DNS takes the IP from Applications Terraform state. The imported A record has TTL 60 and is not proxied; other records remain outside this configuration.

The instance and IP have `prevent_destroy`. `bootstrap_ssh_cidr` defaults to empty. Allow one recovery source temporarily only when needed, then close it. Changing Lightsail public-port rules can replace that firewall resource: review the plan and verify the resulting ports.

## GitHub Actions

Release tags build the image and assume `deploy-planning-poker` through GitHub OIDC. Trust allows this repository's `v*` tags and `master`. Only the deployment job requests `id-token: write`.

| Repository variable | Value |
|---|---|
| `AWS_REGION` | `us-east-1` |
| `AWS_DEPLOY_ROLE_ARN` | `arn:aws:iam::675704851310:role/deploy-planning-poker` |
| `SSM_INSTANCE_ID` | `mi-060db76ff82814ac5` |
| `DOMAIN_NAME` | `pokerplanning.net` |
| `GA_MEASUREMENT_ID` | GA4 web stream measurement ID, injected into the release image |

AWS access-key and SSH-private-key deployment secrets were removed. The role can invoke only `PlanningPoker-Deploy` and `PlanningPoker-Health` on the configured node and read command status. It cannot invoke arbitrary shell documents. Both the client and document validate version/image tags.

`Verify AWS deployment access` can be dispatched to check OIDC and the target's local HTTPS endpoint. Supplying the optional existing version also redeploys it. New-account verification passed in [run 41541720914](https://github.com/damione1/planning-poker/actions/runs/41541720914).

```sh
export AWS_PROFILE=aws-applications-admin AWS_REGION=us-east-1
export SSM_INSTANCE_ID=mi-060db76ff82814ac5
./scripts/ci-deploy-lightsail.sh 0.6.0 0.6.0
```

## Backups and recovery

Lightsail automatic snapshots run at **08:00 UTC**. SSM association `planning-poker-daily-backup` runs at **06:00 UTC**, following the latest `PlanningPoker-Backup` document. It creates SQLite online backups, checks integrity, copies application files and records the running image digest, Compose configuration and Traefik ACME state.

Archives go to private, encrypted, versioned bucket `panetone-planning-poker-backups-912928333407`, prefix `planning-poker/`. The node has **PutObject only**, with no read or delete permission. Current objects and noncurrent versions each have a 90-day lifecycle; older versions can outlive current-object expiry. Archives contain application data and TLS private material: keep access restricted.

Migration restored and integrity-checked both SQLite databases, preserving image `0.6.0` at digest `sha256:13bf3e8510b388bf9a975c3c0570eeb2c10ddebbfa4cc1c01140379a631986dc`. Final frozen-source recovery archive:

```text
s3://panetone-planning-poker-backups-912928333407/planning-poker/20261102T225645Z-ip-172-26-8-229.ec2.internal.tar.gz
SHA256: 67cb960677fcb63a60dc98d2b33a4e7f11d835ec6ca097539e4043ca8eb7dc2a
```

For recovery, prepare a replacement in restore mode, register it in SSM and select a recent archive. `applications/scripts/restore-from-s3.sh` replaces the destination's `pb_data`: use a prepared recovery target. It checks the account, archive paths and SQLite integrity, restores the recorded image digest and tests local HTTPS. Verify the downloaded checksum against backup output before restoring. Grant temporary, prefix-scoped read permissions **on both the node role and central bucket**; these were removed after migration. Remove them again after recovery.

Freeze/back up the current writer before replacing it. Validate the replacement, update its node ID in `applications/deployment.auto.tfvars`, apply Terraform and update GitHub variables, then cut over DNS. Delete the superseded instance and IP after validation to avoid ongoing duplicate charges. Online file copies are not atomic with database backups; stop application writes when an exact common boundary is required.

## Host reconstruction

`lightsail-user-data.sh` bootstraps Docker and Traefik. Applications Terraform sets `PLANNING_POKER_BOOTSTRAP_MODE=restore`, leaving the app stopped until restoration. User-data changes are ignored for the existing host to avoid replacement.

Use the sibling homelab registration helper:

```sh
python3 ../homelab/terraform/aws-account/scripts/register-lightsail-ssm.py --help
```

Pass `--profile aws-applications-admin`, `--expected-account 675704851310`, the instance name and SSM role. The helper creates a single-use activation and removes it afterward. Prefer the host key returned by AWS. The explicit fresh-instance trust option checks the instance age and AWS public IP before pinning its key; review the fingerprint. Close temporary SSH after the node is online.

Architecture, access guidance and remaining personal actions are in the Obsidian AWS hub under `../notes/Projects/AWS/`.
