#!/usr/bin/env python3
"""Check publishable source without reading or echoing local secret values."""
import re,subprocess,sys,xml.etree.ElementTree as E
from pathlib import Path
root=Path(__file__).resolve().parents[1];problems=[]
paths=subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard'],cwd=root,text=True).splitlines()
for name in set(paths):
 p=root/name
 if not p.is_file():continue
 lower=p.name.lower()
 if (lower=='.env' or lower.endswith(('.key','.pem','.p12','.sqlite','.db')) or lower in ('secrets.json','session.json','adbkey','identity.json')) and not name.startswith('third_party/'):
  problems.append(name+': private file type');continue
 if p.stat().st_size>3_000_000:continue
 try:text=p.read_text()
 except (UnicodeError,OSError):continue
 if re.search(r'(?m)^-----BEGIN (?:RSA )?PRIVATE KEY-----$',text):problems.append(name+': private key')
for file in (root/'unraid').glob('*.xml'):
 r=E.parse(file).getroot()
 for c in r.findall('Config'):
  if c.attrib.get('Mask')=='true' and ((c.text or '').strip() or c.attrib.get('Default','').strip()):problems.append(str(file.relative_to(root))+': nonempty secret default')
for required in ('Dockerfile','plugins/maa/Dockerfile','plugins/maa/plugin.json','docs/PLUGINS.md','docs/RELEASING.md','unraid/my-scrcpycat.xml','unraid/my-maa-plugin.xml'):
 if not (root/required).is_file():problems.append(required+': missing')
if problems:
 print('\n'.join(problems));sys.exit(1)
print('Publishable source checks passed; runtime appdata and ignored files are excluded.')
