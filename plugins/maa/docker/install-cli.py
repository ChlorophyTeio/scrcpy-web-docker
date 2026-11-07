import hashlib,os,platform,tarfile,urllib.request,tempfile,shutil
version='0.7.7'
architecture=os.environ.get('TARGETARCH') or {'x86_64':'amd64','aarch64':'arm64'}.get(platform.machine())
assets={'amd64':('x86_64','b582dd79c3f9b42c3a9b4fb6bcf76cc25bfc96600e83028881c97b0c621876a1'),'arm64':('aarch64','dc38518a1b6111bb0b7a8eea63578dacaebc89651bbbc2f7d568c3484e16fb56')}
arch,digest=assets[architecture]
url=f'https://github.com/MaaAssistantArknights/maa-cli/releases/download/v{version}/maa_cli-v{version}-{arch}-unknown-linux-gnu.tar.gz'
with tempfile.TemporaryDirectory() as tmp:
 file=tmp+'/cli.tar.gz'
 with urllib.request.urlopen(url,timeout=120) as response,open(file,'wb') as f:shutil.copyfileobj(response,f)
 with open(file,'rb') as f:assert hashlib.file_digest(f,'sha256').hexdigest()==digest,'CLI SHA256 mismatch'
 with tarfile.open(file) as archive:
  members=[m for m in archive.getmembers() if m.isfile() and m.name.rsplit('/',1)[-1]=='maa'];assert len(members)==1
  with archive.extractfile(members[0]) as source,open('/usr/local/bin/maa','wb') as out:shutil.copyfileobj(source,out)
 os.chmod('/usr/local/bin/maa',0o755)
