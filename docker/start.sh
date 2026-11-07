#!/bin/bash
set -Eeuo pipefail
: "${SCRCPYCAT_ADMIN_PASSWORD:?Set an initial administrator password}"
: "${SCRCPYCAT_POSTGRES_DSN:?Configure PostgreSQL}"
: "${SCRCPYCAT_REDIS_URL:?Configure Redis}"
mkdir -p /data/adb /data/plugins /data/agent /data/agent-state "${SCRCPYCAT_ASSET_DIR:-/data/assets}"
chown 65532:65532 "${SCRCPYCAT_ASSET_DIR:-/data/assets}"
# ADB's key directory is persistent, without changing the web process HOME.
if [[ ! -e /root/.android ]];then
 if [[ -f /data/home/.android/adbkey ]];then ln -s /data/home/.android /root/.android;else ln -s /data/adb /root/.android;fi
fi
pids=()
cleanup(){ trap - EXIT TERM INT; ((${#pids[@]}==0)) || { kill -TERM "${pids[@]}" 2>/dev/null || true;wait "${pids[@]}" 2>/dev/null || true; }; }
trap cleanup EXIT
trap 'exit 0' TERM INT
/opt/adb-usb-supervisor.sh & pids+=("$!")
setpriv --reuid=65532 --regid=65532 --clear-groups /usr/local/bin/scrcpycat-controlplane & pids+=("$!")
if [[ "${SCRCPYCAT_INTERNAL_AGENT_ENABLED:-true}" == true ]];then node /opt/integrated-agent.cjs & pids+=("$!");fi
if [[ "${SCRCPYCAT_TLS_ENABLED:-true}" == true ]];then
 key=${SCRCPYCAT_TLS_KEY:-/data/tls/server.key};cert=${SCRCPYCAT_TLS_CERT:-/data/tls/server.crt}
 if [[ ! -f "$key" || ! -f "$cert" ]];then
  mkdir -p /data/tls
  san=$(node -e "const n=require('net');let names=['DNS:localhost','IP:127.0.0.1'];try{const h=new URL(process.env.SCRCPYCAT_PUBLIC_URL||'https://localhost').hostname.replace(/^\\[|\\]$/g,'');names.push((n.isIP(h)?'IP:':'DNS:')+h)}catch{};console.log([...new Set(names)].join(','))")
  openssl req -x509 -newkey rsa:2048 -nodes -days 3650 -keyout "$key" -out "$cert" -subj /CN=ScrcpyCat -addext "subjectAltName=$san" >/dev/null 2>&1
  chmod 600 "$key"
 fi
 node /opt/tls-proxy.cjs & pids+=("$!")
fi
set +e
wait -n "${pids[@]}"
exit "$?"
