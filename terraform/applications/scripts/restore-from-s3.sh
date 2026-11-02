#!/bin/bash
# Run on the Applications managed node with temporary migration-read permission.
set -euo pipefail
umask 077
archive_uri="${1:?S3 archive URI required}"
test "$(aws sts get-caller-identity --query Account --output text)" = "675704851310"
case "$archive_uri" in
  s3://panetone-planning-poker-backups-912928333407/planning-poker/*.tar.gz) ;;
  *) echo 'Unexpected backup location' >&2; exit 2 ;;
esac
restore_dir=$(mktemp -d /var/tmp/planning-poker-migration.XXXXXXXX)
trap 'rm -rf "$restore_dir"' EXIT
aws s3 cp "$archive_uri" "$restore_dir/archive.tar.gz" --region us-east-1 --only-show-errors
python3 - "$restore_dir" <<'PY'
import json,pathlib,re,sqlite3,subprocess,sys,tarfile
root=pathlib.Path(sys.argv[1]).resolve()
with tarfile.open(root/'archive.tar.gz') as archive:
    for member in archive.getmembers():
        path=(root/member.name).resolve()
        if root not in path.parents or member.issym() or member.islnk():
            raise RuntimeError('Unsafe archive member')
    archive.extractall(root)
databases=list((root/'pb_data').glob('*.db'))
assert databases,'Archive contains no databases'
for db in databases:
    with sqlite3.connect('file:'+str(db)+'?mode=ro',uri=True) as connection:
        assert connection.execute('PRAGMA integrity_check').fetchone()[0]=='ok'
manifest=json.loads((root/'host-config/image.json').read_text())
digest=manifest['image']
assert re.fullmatch(r'ghcr\.io/damione1/planning-poker@sha256:[a-f0-9]{64}',digest)
subprocess.run(['docker','pull',digest],check=True)
subprocess.run(['docker','tag',digest,'ghcr.io/damione1/planning-poker:latest'],check=True)
print('Restored database integrity checks passed:',len(databases))
print('Pinned application image:',digest)
PY
docker rm -f planning-poker >/dev/null 2>&1 || true
rm -rf /mnt/data/pb_data
cp -a "$restore_dir/pb_data" /mnt/data/pb_data
chown -R 1000:1000 /mnt/data/pb_data
install -m 600 -o 1000 -g 1000 "$restore_dir/host-config/docker-compose.prod.yml" /opt/planning-poker/docker-compose.prod.yml
install -m 600 -o 1000 -g 1000 "$restore_dir/host-config/traefik-acme.json" /mnt/data/traefik/acme/acme.json
set -a
. /etc/environment
set +a
cd /opt/planning-poker
docker compose -f docker-compose.prod.yml up -d --no-deps --force-recreate --pull never traefik
docker compose -f docker-compose.prod.yml up -d --no-deps --force-recreate --pull never app
for attempt in $(seq 1 40); do
  if curl -fsS --max-time 5 --resolve pokerplanning.net:443:127.0.0.1 https://pokerplanning.net/monitoring/health; then
    echo
    echo 'Destination application and HTTPS certificate verified.'
    exit 0
  fi
  sleep 2
done
echo 'Destination did not become healthy' >&2
exit 1
