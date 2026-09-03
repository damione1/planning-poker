#!/bin/bash
# GitHub Actions runner: mint a 60s Lightsail SSH cert and run the remote deploy.
set -euo pipefail

INSTANCE_NAME="${LIGHTSAIL_INSTANCE_NAME:-planning-poker}"
VERSION="${1:?version required}"
IMAGE_TAG="${2:-$VERSION}"

DETAILS=$(aws lightsail get-instance-access-details \
    --instance-name "$INSTANCE_NAME" \
    --protocol ssh \
    --output json)

IP=$(echo "$DETAILS" | jq -r .accessDetails.ipAddress)
USER=$(echo "$DETAILS" | jq -r .accessDetails.username)

umask 077
WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT

printf '%s\n' "$(echo "$DETAILS" | jq -r .accessDetails.privateKey)" > "$WORKDIR/id_rsa"
printf '%s\n' "$(echo "$DETAILS" | jq -r .accessDetails.certKey)" > "$WORKDIR/id_rsa-cert.pub"
echo "$DETAILS" | jq -r --arg ip "$IP" '.accessDetails.hostKeys[] | "\($ip) \(.algorithm) \(.publicKey)"' > "$WORKDIR/known_hosts"

echo "Deploying v${VERSION} to ${USER}@${IP} (${INSTANCE_NAME})"

# Certs from GetInstanceAccessDetails expire in ~60s — SSH immediately.
ssh -i "$WORKDIR/id_rsa" \
    -o CertificateFile="$WORKDIR/id_rsa-cert.pub" \
    -o UserKnownHostsFile="$WORKDIR/known_hosts" \
    -o IdentitiesOnly=yes \
    -o StrictHostKeyChecking=yes \
    -o ConnectTimeout=10 \
    "$USER@$IP" \
    "bash /opt/planning-poker/scripts/deploy.sh $(printf '%q' "$VERSION") $(printf '%q' "$IMAGE_TAG")"
