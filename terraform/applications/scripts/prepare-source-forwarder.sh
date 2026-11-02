#!/bin/bash
# Temporary bridge for clients that still cache the old IP during DNS cutover.
set -euo pipefail
test "$(aws sts get-caller-identity --query Account --output text)" = "912928333407"
destination="${1:?Destination IPv4 required}"
[[ "$destination" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]
docker pull nginx:stable-alpine
mkdir -p /var/tmp/planning-poker-migration
cat > /var/tmp/planning-poker-migration/default.conf <<'NGINX_CONFIG_END'
map $http_upgrade $connection_upgrade {
    default upgrade;
    '' close;
}
server {
    listen 80;
    access_log off;
    location / {
        proxy_pass https://DESTINATION_IPV4;
        proxy_ssl_server_name on;
        proxy_ssl_name pokerplanning.net;
        proxy_ssl_verify on;
        proxy_ssl_verify_depth 5;
        proxy_ssl_trusted_certificate /etc/ssl/certs/ca-certificates.crt;
        proxy_http_version 1.1;
        proxy_set_header Host pokerplanning.net;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 3600s;
    }
}
NGINX_CONFIG_END
sed -i "s/DESTINATION_IPV4/$destination/" /var/tmp/planning-poker-migration/default.conf
echo 'Temporary forwarding configuration prepared; production routing unchanged.'
