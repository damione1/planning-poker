#!/bin/bash
# Executed as root by the SSM association. Never prints application data.
set -euo pipefail
umask 077
backup_dir=$(mktemp -d /var/tmp/planning-poker-backup.XXXXXXXX)
trap 'rm -rf "$backup_dir"' EXIT

python3 - "$backup_dir" <<'PY'
import json, pathlib, shutil, sqlite3, subprocess, sys
source = pathlib.Path('/mnt/data/pb_data')
target = pathlib.Path(sys.argv[1]) / 'pb_data'
target.mkdir()
databases = list(source.glob('*.db'))
excluded = {name for db in databases for name in [db.name, db.name+'-wal', db.name+'-shm', db.name+'-journal']}
for item in source.iterdir():
    if item.name in excluded:
        continue
    destination = target / item.name
    if item.is_dir():
        shutil.copytree(item, destination)
    else:
        shutil.copy2(item, destination)
for db in databases:
    with sqlite3.connect('file:'+str(db)+'?mode=ro', uri=True) as src:
        with sqlite3.connect(target/db.name) as dst:
            src.backup(dst, pages=256)
            result = dst.execute('PRAGMA integrity_check').fetchone()[0]
            if result != 'ok':
                raise RuntimeError('SQLite backup integrity check failed for '+db.name)
print('SQLite online backups passed integrity checks for', len(databases), 'databases.')
host = pathlib.Path(sys.argv[1]) / 'host-config'
host.mkdir()
for name, path in [('docker-compose.prod.yml','/opt/planning-poker/docker-compose.prod.yml'),('traefik-acme.json','/mnt/data/traefik/acme/acme.json')]:
    if pathlib.Path(path).is_file():
        shutil.copy2(path,host/name)
container = json.loads(subprocess.check_output(['docker','inspect','planning-poker']))[0]
image = json.loads(subprocess.check_output(['docker','image','inspect',container['Image']]))[0]
digests = [d for d in image.get('RepoDigests',[]) if d.startswith('ghcr.io/damione1/planning-poker@sha256:')]
if not digests:
    raise RuntimeError('The application image has no reproducible registry digest')
(host/'image.json').write_text(json.dumps({'image':digests[0],'version':container['Config'].get('Labels',{}).get('org.opencontainers.image.version')}))
PY

archive="$backup_dir/planning-poker.tar.gz"
tar -C "$backup_dir" -czf "$archive" pb_data host-config
timestamp=$(date -u +%Y%m%dT%H%M%SZ)
destination="s3://panetone-planning-poker-backups-912928333407/planning-poker/${timestamp}-$(hostname).tar.gz"
aws s3 cp "$archive" "$destination" --region us-east-1 --sse AES256 --no-progress --only-show-errors
printf 'Uploaded encrypted application backup: %s\n' "$destination"
sha256sum "$archive" | cut -d ' ' -f1
