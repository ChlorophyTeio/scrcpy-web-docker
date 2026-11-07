#!/usr/bin/env python3
"""Generate shareable Unraid templates, never by inspecting a live container."""
import argparse,re,xml.etree.ElementTree as E
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--owner',default='ChlorophyTeio');p.add_argument('--repo',default='scrcpy-web-docker');p.add_argument('--image',default='chlorophyimo/scrcpycat');p.add_argument('--maa-image',default='chlorophyimo/scrcpycat-maa');p.add_argument('--cache-image',default='chlorophyimo/scrcpycat-cache');p.add_argument('--output',default='unraid');a=p.parse_args()
if not all(re.fullmatch(r'[A-Za-z0-9_./-]+',v) for v in (a.owner,a.repo,a.image,a.maa_image,a.cache_image)):raise SystemExit('Invalid publication identifier')
base=f'https://github.com/{a.owner}/{a.repo}';raw=f'https://raw.githubusercontent.com/{a.owner}/{a.repo}/main';out=Path(a.output);out.mkdir(parents=True,exist_ok=True)
def config(r,name,target,kind,default='',mode='',required=True,secret=False,description=''):
 E.SubElement(r,'Config',Name=name,Target=target,Default=default,Mode=mode,Type=kind,Display='always',Required=str(required).lower(),Mask=str(secret).lower(),Description=description).text=default
for maa in (False,True):
 r=E.Element('Container',version='2');name='maa-web' if maa else 'scrcpy-web';image=a.maa_image if maa else a.image
 for k,v in {'Name':name,'Repository':image+':latest','Registry':'https://hub.docker.com/r/'+image,'Network':'scrcpycat','Shell':'bash','Privileged':'false','Support':base+'/issues','Project':base,'Category':'Tools:','Icon':raw+('/plugins/maa/icon.svg' if maa else '/frontend/web-app/public/favicon.svg'),'WebUI':'https://[IP]:[PORT:9443]/'+('plugins/maa' if maa else ''),'Overview':('可选 MAA 插件。启动并就绪后自动出现在 ScrcpyCat；首次启动下载核心资源。无独立公开网页端口。' if maa else '网页、USB ADB 和手机 agent 合并容器。PostgreSQL 17、Redis 为外部依赖；MAA 为可选插件。'),'ExtraParams':('--restart=unless-stopped --stop-timeout=40' if maa else "--restart=unless-stopped --stop-timeout=30 --device-cgroup-rule='c 189:* rwm'")}.items():E.SubElement(r,k).text=v
 if maa:
  # Plugin WebUI belongs to the main container; no nonexistent 9443 port on this worker.
  r.find('WebUI').text=''
  for title,target,default in [('配置','/app/config','/mnt/user/appdata/scrcpycat-maa/config'),('核心与资源','/root/.local/share/maa','/mnt/user/appdata/scrcpycat-maa/data'),('下载缓存','/root/.cache/maa','/mnt/user/appdata/scrcpycat-maa/cache')]:config(r,title,target,'Path',default,'rw')
  config(r,'共享 ADB','ADB_SERVER_SOCKET','Variable','tcp:scrcpy-web:5037');config(r,'手机 ADB 序列号','MAA_DEVICE','Variable','',required=True,description='填写 ScrcpyCat 在线设备标识。')
 else:
  config(r,'HTTPS 网页','9443','Port','9015','tcp');config(r,'WebRTC UDP','40000-40100','Port','40000-40100','udp')
  config(r,'持久化数据','/data','Path','/mnt/user/appdata/scrcpycat','rw');config(r,'USB 热插拔','/dev/bus/usb','Path','/dev/bus/usb','rw')
  for name,default,secret,description in [('SCRCPYCAT_PUBLIC_URL','',False,'完整网页访问地址，例如 https://主机地址:9015'),('SCRCPYCAT_WEBRTC_HOST_IPS','',False,'Unraid 的局域网 IP，逗号分隔多个地址。'),('SCRCPYCAT_POSTGRES_DSN','',True,'现有 PostgreSQL 17 的专用数据库连接。'),('SCRCPYCAT_REDIS_URL','',False,'同网络 Redis 地址。'),('SCRCPYCAT_ADMIN_USERNAME','admin',False,''),('SCRCPYCAT_ADMIN_PASSWORD','',True,'首次管理员密码，自行填写。'),('SCRCPYCAT_JWT_SECRET','',True,'自行生成并持久保留的随机长字符串。')]:config(r,name,name,'Variable',default,secret=secret,description=description)
  config(r,'插件服务地址','SCRCPYCAT_MAA_URL','Variable','http://maa-web:8080',required=False)
 E.indent(r);E.ElementTree(r).write(out/('my-'+('maa-plugin' if maa else 'scrcpycat')+'.xml'),encoding='utf-8',xml_declaration=True)

r=E.Element('Container',version='2')
cache_image=a.cache_image
for k,v in {'Name':'scrcpycat-cache','Repository':cache_image+':latest','Registry':'https://hub.docker.com/r/'+cache_image,'Network':'scrcpycat','Shell':'sh','Privileged':'false','Support':base+'/issues','Project':base,'Category':'Tools:','Icon':raw+'/cache/icon.svg','WebUI':'','Overview':'ScrcpyCat 内部消息总线，Redis 7.4.12，启用 AOF 持久化和健康检查。与主容器使用同一自定义网络，不映射宿主机端口。','ExtraParams':'--restart=unless-stopped'}.items():E.SubElement(r,k).text=v
config(r,'缓存持久化','/data','Path','/mnt/user/appdata/scrcpycat-cache','rw')
E.indent(r);E.ElementTree(r).write(out/'my-scrcpycat-cache.xml',encoding='utf-8',xml_declaration=True)
print('Generated generic templates in',out)
