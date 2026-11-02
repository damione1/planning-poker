#!/bin/bash
# Permanent IAM key bootstrap was retired in favor of Terraform and GitHub OIDC.
set -euo pipefail
cat >&2 <<'MESSAGE'
The legacy AWS access-key setup is retired.

Account IAM, GitHub OIDC and Systems Manager documents are managed in:
  ../homelab/terraform/aws-account/

Production Lightsail resources are managed in:
  terraform/

See terraform/DEPLOY.md for temporary SSO access and deployment instructions.
MESSAGE
exit 2
