#!/usr/bin/env python3
"""Generate private deployment values locally; never print credentials."""
import argparse
import os
import secrets
from pathlib import Path

p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--output', default='deploy/.env')
p.add_argument('--host-ip', default='127.0.0.1')
p.add_argument('--public-url', default='https://localhost:9015')
p.add_argument('--database', choices=('managed', 'existing'), default='managed')
a = p.parse_args()
path = Path(a.output)
path.parent.mkdir(parents=True, exist_ok=True)
database = secrets.token_hex(24)
values = {
    'COMPOSE_PROFILES': 'database' if a.database == 'managed' else '',
    'SCRCPYCAT_VERSION': '0.1.4',
    'SCRCPYCAT_IMAGE': 'chlorophyimo/scrcpycat:0.1.4',
    'SCRCPYCAT_MAA_IMAGE': 'chlorophyimo/scrcpycat-maa:0.1.4',
    'SCRCPYCAT_CACHE_IMAGE': 'chlorophyimo/scrcpycat-cache:0.1.4',
    'SCRCPYCAT_NETWORK': 'scrcpycat',
    'SCRCPYCAT_EXTERNAL_NETWORK': 'false',
    'SCRCPYCAT_PUBLIC_URL': a.public_url,
    'SCRCPYCAT_HTTPS_PORT': '9015',
    'SCRCPYCAT_ADMIN_USERNAME': 'admin',
    'SCRCPYCAT_ADMIN_PASSWORD': secrets.token_urlsafe(24),
    'SCRCPYCAT_JWT_SECRET': secrets.token_urlsafe(48),
    'SCRCPYCAT_POSTGRES_PASSWORD': database,
    'SCRCPYCAT_POSTGRES_DSN': (
        f'postgres://scrcpycat:{database}@postgres:5432/scrcpycat?sslmode=disable'
        if a.database == 'managed' else ''
    ),
    'SCRCPYCAT_WEBRTC_HOST_IPS': a.host_ip,
    'SCRCPYCAT_WEBRTC_UDP_MIN': '40000',
    'SCRCPYCAT_WEBRTC_UDP_MAX': '40100',
    'MAA_DEVICE': '',
}
for v in values.values():
    # Compose expands dollars and interprets quotes/comments in unquoted .env values.
    if any(c in v for c in '\n\r$#\'"') or v != v.strip():
        raise SystemExit('Invalid value; use plain host/IP/URL arguments, then edit the private .env if needed')
try:
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
except FileExistsError:
    raise SystemExit(f'{path} already exists; leaving its credentials unchanged')
with os.fdopen(fd, 'w') as f:
    f.write(''.join(k + '=' + v + '\n' for k, v in values.items()))
print(f'Created {path} with mode 0600. Open it locally to view the initial login or customize deployment.')
if a.database == 'existing':
    print('Fill SCRCPYCAT_POSTGRES_DSN before starting. Use a shared external Docker network for a container database.')
