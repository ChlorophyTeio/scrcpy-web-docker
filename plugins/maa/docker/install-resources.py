"""Update plugin-owned task definitions without discarding additional user tasks."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import tempfile


def install_resources(source, target):
    source, target = Path(source), Path(target)
    bundled = json.loads(source.read_text())
    before = target.read_bytes() if target.exists() else None
    existing = json.loads(before) if before is not None else {}
    if not isinstance(bundled, dict) or not isinstance(existing, dict):
        raise ValueError('MAA task resources must be JSON objects')
    merged = {**existing, **bundled}
    if before is not None and merged == existing:
        return False
    target.parent.mkdir(parents=True, exist_ok=True)
    if before is not None:
        backup_dir = target.parents[2] / 'resource-backups'
        backup_dir.mkdir(parents=True, exist_ok=True)
        backup = backup_dir / ('web-mall-' + hashlib.sha256(before).hexdigest() + '.json')
        if not backup.exists():
            fd = os.open(backup, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            with os.fdopen(fd, 'wb') as f:
                f.write(before)
    fd, temporary = tempfile.mkstemp(prefix='.web-mall-', suffix='.tmp', dir=target.parent)
    try:
        with os.fdopen(fd, 'w') as f:
            json.dump(merged, f, ensure_ascii=False, indent=2)
            f.write('\n')
        os.replace(temporary, target)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)
    return True


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', default='/opt/maa-plugin/resources/tasks/web-mall.json')
    parser.add_argument('--target', default=str(Path(os.environ.get('MAA_CONFIG_DIR', '/app/config/maa')) / 'resource/tasks/web-mall.json'))
    args = parser.parse_args()
    if install_resources(args.source, args.target):
        print('Updated bundled MAA task resources; previous definitions backed up when present.')
