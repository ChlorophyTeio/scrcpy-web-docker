#!/bin/bash
set -Eeuo pipefail
mkdir -p /app/config/maa/{profiles,tasks,infrast,resource/tasks}
profile=/app/config/maa/profiles/default.toml
if [[ ! -f "$profile" ]];then
 serial=${MAA_DEVICE:-}
 if [[ -z "$serial" ]];then serial=$(/opt/android/adb devices | awk 'NR>1 && $2=="device" {s=$1;n++} END {if(n==1)print s}');fi
 [[ "$serial" =~ ^[a-zA-Z0-9_.:-]*$ ]] || { echo 'Invalid MAA_DEVICE' >&2;exit 1; }
 cat > "$profile" <<PROFILE
[connection]
adb_path = "/opt/android/adb"
device = "$serial"
config = "CompatPOSIXShell"
[resource]
user_resource = true
[static_options]
cpu_ocr = true
[instance_options]
touch_mode = "Adb"
kill_adb_on_exit = false
PROFILE
fi
python3 /opt/configure-device.py
if [[ ! -f /app/config/maa/cli.toml ]];then printf '[core]\nchannel = "Stable"\n[resource]\nauto_update = false\n' > /app/config/maa/cli.toml;fi
python3 /opt/install-resources.py
if [[ ! -f /root/.local/share/maa/lib/libMaaCore.so ]];then maa install stable --batch --test-time 0;fi
maa version
/usr/local/bin/maa-phone-display restore || true
/app/main & app_pid=$!
cleanup(){ trap - EXIT TERM INT;kill -TERM "$app_pid" 2>/dev/null || true;wait "$app_pid" 2>/dev/null || true;/usr/local/bin/maa-phone-display restore || true; }
trap cleanup EXIT
trap 'exit 0' TERM INT
set +e
wait "$app_pid"
exit "$?"
