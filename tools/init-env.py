#!/usr/bin/env python3
"""Generate private deployment values locally; never print credentials."""
import argparse,secrets,os
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--output',default='deploy/.env');p.add_argument('--host-ip',default='127.0.0.1');p.add_argument('--public-url',default='https://localhost:9015');a=p.parse_args()
path=Path(a.output);path.parent.mkdir(parents=True,exist_ok=True);password=secrets.token_urlsafe(24);database=secrets.token_hex(24)
values={'SCRCPYCAT_VERSION':'0.1.0','SCRCPYCAT_IMAGE':'chlorophyimo/scrcpycat:0.1.0','SCRCPYCAT_MAA_IMAGE':'chlorophyimo/scrcpycat-maa:0.1.0','SCRCPYCAT_PUBLIC_URL':a.public_url,'SCRCPYCAT_HTTPS_PORT':'9015','SCRCPYCAT_ADMIN_USERNAME':'admin','SCRCPYCAT_ADMIN_PASSWORD':password,'SCRCPYCAT_JWT_SECRET':secrets.token_urlsafe(48),'SCRCPYCAT_POSTGRES_PASSWORD':database,'SCRCPYCAT_POSTGRES_DSN':f'postgres://scrcpycat:{database}@postgres:5432/scrcpycat?sslmode=disable','SCRCPYCAT_WEBRTC_HOST_IPS':a.host_ip,'SCRCPYCAT_WEBRTC_UDP_MIN':'40000','SCRCPYCAT_WEBRTC_UDP_MAX':'40100','MAA_DEVICE':''}
for v in values.values():
 if '\n' in v or '\r' in v:raise SystemExit('Invalid multiline value')
fd=os.open(path,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
with os.fdopen(fd,'w') as f:f.write(''.join(k+'='+v+'\n' for k,v in values.items()))
print(f'Created {path} with mode 0600. Open it locally to view the initial login or customize deployment.')
