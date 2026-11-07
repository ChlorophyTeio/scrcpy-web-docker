"""Apply an explicit device selection without rewriting other CLI settings."""
import os,re,json,tempfile
from pathlib import Path
def update_device(path,serial):
 if not serial:return
 if not re.fullmatch(r'[A-Za-z0-9_.:\[\]-]{1,256}',serial):raise ValueError('Invalid MAA_DEVICE')
 path=Path(path);original=path.read_text();line='device = '+json.dumps(serial)
 pattern=r'(?ms)(^\[connection\][ \t]*\n)(.*?)(?=^\[|\Z)'
 def replace(match):
  body=match[2]
  if re.search(r'(?m)^\s*device\s*=',body):body=re.sub(r'(?m)^[ \t]*device\s*=.*$',line,body,count=1)
  else:body=line+'\n'+body
  return match[1]+body
 updated=re.sub(pattern,replace,original,count=1)
 if updated==original and not re.search(pattern,original):updated+='\n[connection]\n'+line+'\n'
 if updated==original:return
 fd,tmp=tempfile.mkstemp(prefix='.device-',dir=path.parent)
 try:
  with os.fdopen(fd,'w') as f:f.write(updated)
  os.replace(tmp,path)
 finally:
  if os.path.exists(tmp):os.unlink(tmp)
if __name__=='__main__':update_device('/app/config/maa/profiles/default.toml',os.getenv('MAA_DEVICE',''))
