"""Check credential generation and refusal to overwrite an existing deployment."""
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import unittest

SCRIPT = Path(__file__).with_name('init-env.py')

class InitEnvTest(unittest.TestCase):
    def test_managed_secrets_and_existing_file(self):
        with tempfile.TemporaryDirectory() as d:
            dest = Path(d) / '.env'
            cmd = [sys.executable, str(SCRIPT), '--output', str(dest)]
            result = subprocess.run(cmd, capture_output=True, text=True, check=True)
            before = dest.read_bytes()
            values = dict(line.split('=', 1) for line in before.decode().splitlines())
            self.assertEqual(stat.S_IMODE(dest.stat().st_mode), 0o600)
            self.assertEqual(values['COMPOSE_PROFILES'], 'database')
            self.assertIn(values['SCRCPYCAT_POSTGRES_PASSWORD'], values['SCRCPYCAT_POSTGRES_DSN'])
            for key in ('SCRCPYCAT_ADMIN_PASSWORD', 'SCRCPYCAT_JWT_SECRET', 'SCRCPYCAT_POSTGRES_PASSWORD'):
                self.assertGreaterEqual(len(values[key]), 32)
                self.assertNotIn(values[key], result.stdout + result.stderr)
            self.assertNotEqual(subprocess.run(cmd, capture_output=True).returncode, 0)
            self.assertEqual(before, dest.read_bytes())

    def test_existing_database_requires_configuration(self):
        with tempfile.TemporaryDirectory() as d:
            dest = Path(d) / '.env'
            subprocess.run([sys.executable, str(SCRIPT), '--output', str(dest), '--database', 'existing'], capture_output=True, check=True)
            values = dict(line.split('=', 1) for line in dest.read_text().splitlines())
            self.assertEqual(values['COMPOSE_PROFILES'], '')
            self.assertEqual(values['SCRCPYCAT_POSTGRES_DSN'], '')

    def test_rejects_compose_expansion_in_arguments(self):
        with tempfile.TemporaryDirectory() as d:
            dest = Path(d) / '.env'
            result = subprocess.run([sys.executable, str(SCRIPT), '--output', str(dest), '--public-url', 'https://example.com/$BAD'], capture_output=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(dest.exists())
