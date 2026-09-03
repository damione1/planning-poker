#!/bin/bash
# Runs on the Lightsail instance. Pulls a GHCR tag and recreates only the app
# container so Traefik/certs stay up.
set -euo pipefail

VERSION="${1:-latest}"
IMAGE_TAG="${2:-$VERSION}"
IMAGE="ghcr.io/damione1/planning-poker"
COMPOSE="/opt/planning-poker/docker-compose.prod.yml"

echo "=== Deploying Planning Poker $VERSION (image tag: $IMAGE_TAG) ==="

if [ -f /etc/environment ]; then
    set -a
    # shellcheck disable=SC1091
    source /etc/environment
    set +a
fi

if [ -z "${DOMAIN_NAME:-}" ] || [ -z "${LETS_ENCRYPT_EMAIL:-}" ]; then
    echo "Required environment variables not set (DOMAIN_NAME / LETS_ENCRYPT_EMAIL)"
    exit 1
fi

cd /opt/planning-poker

echo "Pulling ${IMAGE}:${IMAGE_TAG}"
docker pull "${IMAGE}:${IMAGE_TAG}"

if [ "$IMAGE_TAG" != "latest" ]; then
    docker tag "${IMAGE}:${IMAGE_TAG}" "${IMAGE}:latest"
fi

echo "Recreating app container (Traefik left running)"
docker compose -f "$COMPOSE" up -d --no-deps --force-recreate app

echo "Waiting for container to become healthy..."
ok=0
for i in $(seq 1 36); do
    status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' planning-poker 2>/dev/null || echo "missing")
    echo "  health=${status} (${i})"
    if [ "$status" = "healthy" ]; then
        ok=1
        break
    fi
    if [ "$status" = "missing" ] || [ "$status" = "exited" ] || [ "$status" = "dead" ]; then
        docker compose -f "$COMPOSE" logs --tail 80 app
        exit 1
    fi
    sleep 5
done

if [ "$ok" != 1 ]; then
    echo "Container did not become healthy"
    docker compose -f "$COMPOSE" logs --tail 80 app
    exit 1
fi

echo "Deployment successful"
docker compose -f "$COMPOSE" ps
