// A local Agent shares USB ADB with the web service. Only scoped deployment
// credentials and media settings are passed to the child, never DB/admin secrets.
const fs=require('fs'),path=require('path'),{spawn}=require('child_process');
const tokenFile=process.env.SCRCPYCAT_DEPLOYMENT_TOKEN_FILE||'/data/agent/deployment-token';
let child,stopping=false;const sleep=ms=>new Promise(r=>setTimeout(r,ms));
for(const sig of ['SIGTERM','SIGINT'])process.on(sig,()=>{stopping=true;child?.kill('SIGTERM')});
async function request(endpoint,body,token){const r=await fetch('http://127.0.0.1:8443'+endpoint,{method:body===undefined?'GET':'POST',headers:{'Content-Type':'application/json',...(token?{Authorization:'Bearer '+token}:{})},body:body===undefined?undefined:JSON.stringify(body),signal:AbortSignal.timeout(5000)});if(!r.ok)throw Error('local service returned '+r.status);return r.json()}
async function token(){
 if(process.env.SCRCPYCAT_DEPLOYMENT_TOKEN)return process.env.SCRCPYCAT_DEPLOYMENT_TOKEN;
 if(fs.existsSync(tokenFile))return fs.readFileSync(tokenFile,'utf8').trim();
 const auth=await request('/api/login',{username:process.env.SCRCPYCAT_ADMIN_USERNAME||'admin',password:process.env.SCRCPYCAT_ADMIN_PASSWORD||''});
 const result=await request('/api/admin/deployment-credentials',{name:'integrated-usb-agent',ttl_seconds:0},auth.token);
 fs.mkdirSync(path.dirname(tokenFile),{recursive:true,mode:0o700});fs.writeFileSync(tokenFile,result.token,{mode:0o600});return result.token;
}
(async()=>{while(!stopping){try{
 await request('/healthz');const credential=await token();if(stopping)break;
 const env={PATH:'/usr/local/bin:/usr/bin:/bin',SCRCPYCAT_DEPLOYMENT_TOKEN:credential};
 for(const key of ['SCRCPYCAT_WEBRTC_HOST_IPS','SCRCPYCAT_WEBRTC_UDP_MIN','SCRCPYCAT_WEBRTC_UDP_MAX'])if(process.env[key])env[key]=process.env[key];
 child=spawn('/usr/local/bin/scrcpycat-adb-deployer',['--mode=host','--adb-server=127.0.0.1:5037','--adb=/bin/false','--state-dir='+ (process.env.SCRCPYCAT_AGENT_STATE_DIR||'/var/lib/scrcpycat/adb-deployer'),'--signaling=ws://127.0.0.1:8443/register_agent','--control-plane=http://127.0.0.1:8443'],{stdio:'inherit',env});
 await new Promise((resolve,reject)=>{child.once('exit',resolve);child.once('error',reject)});child=null;
 }catch(e){if(!stopping)console.error('[integrated-agent] waiting for local service/configuration:',e.message)}
 if(!stopping)await sleep(3000);
}})().catch(()=>process.exit(1));
