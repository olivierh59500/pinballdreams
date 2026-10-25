#!/bin/sh
# Build, install and launch on one selected Android device.
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$script_dir/android-env.sh"
if [ "$#" -ne 0 ]; then
    echo 'Usage: ./scripts/run-android.sh' >&2
    exit 2
fi
adb_bin=${ADB:-"$android_sdk/platform-tools/adb"}
serial=${ANDROID_SERIAL:-}
if [ -z "$serial" ]; then
    devices=$("$adb_bin" devices)
    serial=$(printf '%s\n' "$devices" | awk '$2 == "device" { print $1 }')
    count=$(printf '%s\n' "$serial" | awk 'NF { n++ } END { print n+0 }')
    [ "$count" -eq 1 ] || { echo 'Set ANDROID_SERIAL to select one authorized device.' >&2; exit 1; }
fi
"$script_dir/build-android.sh"
"$adb_bin" -s "$serial" install -r "$apk_path"
"$adb_bin" -s "$serial" shell am start -n com.olivierh.pinballdreams/.MainActivity
