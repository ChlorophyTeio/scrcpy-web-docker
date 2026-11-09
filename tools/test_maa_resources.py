import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('maa_resources', ROOT/'plugins/maa/docker/install-resources.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class MAAResourcesTest(unittest.TestCase):
    def test_existing_install_gets_fix_and_keeps_extra_tasks_and_backup(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory)/'maa/resource/tasks/web-mall.json'
            target.parent.mkdir(parents=True)
            before = json.dumps({'WebMall-OperatorSkip': {'baseTask': 'RecruitSkip'}, 'UserTask': {'algorithm': 'JustReturn'}}).encode()
            target.write_bytes(before)
            source = ROOT/'plugins/maa/resources/tasks/web-mall.json'
            self.assertTrue(module.install_resources(source, target))
            data = json.loads(target.read_text())
            self.assertEqual(data['WebMall-OperatorSkip']['template'], 'RecruitSkip.png')
            self.assertIn('UserTask', data)
            backups = list((Path(directory)/'maa/resource-backups').glob('*.json'))
            self.assertEqual(len(backups), 1)
            self.assertEqual(backups[0].read_bytes(), before)
            timestamp = target.stat().st_mtime_ns
            self.assertFalse(module.install_resources(source, target))
            self.assertEqual(target.stat().st_mtime_ns, timestamp)

    def test_invalid_existing_data_is_left_intact(self):
        with tempfile.TemporaryDirectory() as directory:
            target=Path(directory)/'maa/resource/tasks/web-mall.json'
            target.parent.mkdir(parents=True); target.write_text('broken JSON')
            with self.assertRaises(json.JSONDecodeError):
                module.install_resources(ROOT/'plugins/maa/resources/tasks/web-mall.json', target)
            self.assertEqual(target.read_text(), 'broken JSON')
