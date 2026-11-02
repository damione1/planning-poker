#!/bin/bash
# Deploy through a restricted Systems Manager document; no inbound SSH needed.
set -euo pipefail

INSTANCE_ID="${SSM_INSTANCE_ID:?SSM_INSTANCE_ID must name the registered Planning Poker node}"
VERSION="${1:?version required}"
IMAGE_TAG="${2:-$VERSION}"

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$ ]] ||
   [[ ! "$IMAGE_TAG" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$ ]]; then
    echo "Invalid version or image tag" >&2
    exit 2
fi

PARAMETERS=$(jq -nc --arg version "$VERSION" --arg tag "$IMAGE_TAG" '{Version:[$version],ImageTag:[$tag]}')
COMMAND_ID=$(aws ssm send-command \
    --instance-ids "$INSTANCE_ID" --document-name PlanningPoker-Deploy \
    --parameters "$PARAMETERS" --timeout-seconds 600 \
    --comment "Planning Poker deployment $VERSION" --query Command.CommandId --output text)

echo "Deployment command $COMMAND_ID submitted to $INSTANCE_ID"
for ((attempt=1; attempt<=150; attempt++)); do
    if ! RESULT=$(aws ssm get-command-invocation \
        --command-id "$COMMAND_ID" --instance-id "$INSTANCE_ID" --output json 2>&1); then
        if [[ "$RESULT" == *InvocationDoesNotExist* ]]; then
            sleep 5
            continue
        fi
        printf '%s\n' "$RESULT" >&2
        exit 1
    fi
    STATUS=$(jq -r .Status <<< "$RESULT")
    case "$STATUS" in
        Success) jq -r '.StandardOutputContent, .StandardErrorContent' <<< "$RESULT"; exit 0 ;;
        Failed|Cancelled|TimedOut|Cancelling)
            jq -r '.StandardOutputContent, .StandardErrorContent, .StatusDetails' <<< "$RESULT" >&2
            exit 1 ;;
        Pending|InProgress|Delayed) sleep 5 ;;
        *) printf 'Unexpected SSM command status: %s\n' "$STATUS" >&2; exit 1 ;;
    esac
done
echo "Timed out waiting for deployment command $COMMAND_ID" >&2
exit 1
