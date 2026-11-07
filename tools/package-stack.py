#!/usr/bin/env python3
"""Package a standalone image-only Compose stack; never include local .env files."""
import argparse
import io
import re
import tarfile
from pathlib import Path

root = Path(__file__).resolve().parents[1]
p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--version', required=True)
p.add_argument('--output', default='dist')
a = p.parse_args()
if not re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?', a.version):
    p.error('Invalid version')
sources = {
    'compose.yml': root / 'deploy/docker-compose.yml',
    '.env.example': root / 'deploy/.env.example',
    'init-env.py': root / 'tools/init-env.py',
    'README.md': root / 'docs/STACK.md',
    'LICENSE': root / 'LICENSE',
}
# The version is pinned in all generated defaults. Avoid mislabelling a later bundle.
for filename in ('compose.yml', '.env.example', 'init-env.py'):
    text = sources[filename].read_text()
    for image in ('scrcpycat', 'scrcpycat-maa', 'scrcpycat-cache'):
        if 'chlorophyimo/' + image + ':' + a.version not in text:
            p.error(f'{filename} does not reference the requested image version')
out = Path(a.output)
out.mkdir(parents=True, exist_ok=True)
dest = out / f'scrcpycat-stack-{a.version}.tar.gz'
prefix = f'scrcpycat-stack-{a.version}/'
with tarfile.open(dest, 'w:gz') as archive:
    for name, source in sources.items():
        content = source.read_bytes()
        if name == 'README.md':
            content = content.replace(b'(../cache/SOURCE.md)', b'(https://github.com/ChlorophyTeio/scrcpy-web-docker/blob/main/cache/SOURCE.md)')
        info = tarfile.TarInfo(prefix + name)
        info.size = len(content)
        info.mode = 0o755 if name.endswith('.py') else 0o644
        archive.addfile(info, io.BytesIO(content))
print(dest)
