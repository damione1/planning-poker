#!/bin/bash
set -euo pipefail
test "$(aws sts get-caller-identity --query Account --output text)" = "912928333407"
network=$(python3 -c 'import json,subprocess; print(next(iter(json.loads(subprocess.check_output(["docker","inspect","traefik"]))[0]["NetworkSettings"]["Networks"])))')
docker rm -f planning-poker-migration-proxy >/dev/null 2>&1 || true
docker run -d --name planning-poker-migration-proxy --restart unless-stopped \
  --network "$network" \
  -v /var/tmp/planning-poker-migration/default.conf:/etc/nginx/conf.d/default.conf:ro \
  -v /etc/pki/tls/certs/ca-bundle.crt:/etc/ssl/certs/ca-certificates.crt:ro \
  --label traefik.enable=true \
  --label "traefik.docker.network=$network" \
  --label 'traefik.http.routers.migration.rule=Host(`pokerplanning.net`)' \
  --label traefik.http.routers.migration.entrypoints=websecure \
  --label traefik.http.routers.migration.tls=true \
  --label traefik.http.routers.migration.tls.certresolver=letsencrypt \
  --label traefik.http.routers.migration.priority=200 \
  --label traefik.http.services.migration.loadbalancer.server.port=80 \
  nginx:stable-alpine
echo 'Old-IP forwarding now targets the restored Applications instance.'
