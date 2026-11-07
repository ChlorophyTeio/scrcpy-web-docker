#!/bin/bash
# Restart only the shared ADB process if its USB discovery misses an attached phone.
set -uo pipefail
adb_bin=${ADB_USB_BINARY:-/data/dependencies/adb/adb}
sysfs=${ADB_USB_SYSFS_ROOT:-/sys/bus/usb/devices}
usbfs=${ADB_USB_DEVICE_ROOT:-/dev/bus/usb}
interval=${ADB_USB_CHECK_INTERVAL:-10}
threshold=${ADB_USB_MISS_THRESHOLD:-3}
cooldown=${ADB_USB_RECOVERY_COOLDOWN:-120}
adb_pid= pause_pid= stop=0
log() { printf '%s [usb-adb] %s\n' "$(date -u +%FT%TZ)" "$*" >&2; }
physical_adb_present() {
    local iface base class subclass protocol bus dev authorized node
    for iface in "$sysfs"/*:*; do
        [[ -d "$iface" ]] || continue
        read -r class < "$iface/bInterfaceClass" 2>/dev/null || continue
        read -r subclass < "$iface/bInterfaceSubClass" 2>/dev/null || continue
        read -r protocol < "$iface/bInterfaceProtocol" 2>/dev/null || continue
        [[ "$class:$subclass:$protocol" == ff:42:01 ]] || continue
        base=${iface%%:*}
        read -r authorized < "$base/authorized" 2>/dev/null || continue
        [[ "$authorized" == 1 ]] || continue
        read -r bus < "$base/busnum" 2>/dev/null || continue
        read -r dev < "$base/devnum" 2>/dev/null || continue
        [[ "$bus" =~ ^[0-9]+$ && "$dev" =~ ^[0-9]+$ ]] || continue
        printf -v node '%s/%03d/%03d' "$usbfs" "$((10#$bus))" "$((10#$dev))"
        [[ -c "$node" && -r "$node" && -w "$node" ]] && return 0
    done
    return 1
}
adb_inventory_empty() {
    local inventory serial state rest header=0
    inventory=$(timeout 5 "$adb_bin" -P 5037 devices 2>/dev/null) || return 1
    while read -r serial state rest; do
        [[ "$serial" == List && "$state" == of ]] && { header=1; continue; }
        # Offline/unauthorized devices must not trigger a global restart either.
        [[ -n "$serial" ]] && return 1
    done <<< "$inventory"
    [[ "$header" == 1 ]]
}
start_adb() {
    "$adb_bin" -a -P 5037 server nodaemon &
    adb_pid=$!
    log "shared ADB started"
}
stop_adb() {
    [[ -n "$adb_pid" ]] || return 0
    if kill -0 "$adb_pid" 2>/dev/null; then
        kill -TERM "$adb_pid" 2>/dev/null || true
        for ((n=0;n<20;n++)); do
            kill -0 "$adb_pid" 2>/dev/null || break
            sleep 0.1
        done
        kill -KILL "$adb_pid" 2>/dev/null || true
    fi
    wait "$adb_pid" 2>/dev/null || true
    adb_pid=
}
cleanup() {
    trap - EXIT TERM INT
    [[ -z "$pause_pid" ]] || kill "$pause_pid" 2>/dev/null || true
    stop_adb
}
main() {
    local misses=0 last_recovery=-100000 next_start=0 now
    trap cleanup EXIT
    trap 'stop=1' TERM INT
    start_adb
    while ((stop==0)); do
        sleep "$interval" & pause_pid=$!
        wait "$pause_pid" 2>/dev/null || true
        pause_pid=
        ((stop==0)) || break
        now=$SECONDS
        if ! kill -0 "$adb_pid" 2>/dev/null; then
            stop_adb
            if ((now>=next_start)); then
                log "ADB exited; restarting shared ADB"
                start_adb
                next_start=$((now+10))
            else
                # Let the container's existing supervisor handle repeated crashes.
                log "ADB repeatedly exited; requesting service restart"
                return 1
            fi
            misses=0
            continue
        fi
        if physical_adb_present && adb_inventory_empty; then
            ((misses+=1))
        else
            misses=0
        fi
        if ((misses>=threshold && now-last_recovery>=cooldown)); then
            log "USB debugging interface present but ADB inventory stayed empty; rescanning"
            stop_adb
            start_adb
            last_recovery=$now
            misses=0
        fi
    done
}
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then main; fi
