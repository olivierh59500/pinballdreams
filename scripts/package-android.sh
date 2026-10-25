#!/bin/sh
# Package the existing Go AAR with SDK tools when Gradle IPC is unavailable.
# The conventional Gradle application remains the primary Android build.
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$script_dir/android-env.sh"
cd "$project_root"
package_dir="$android_cache/package"
tools_dir="$android_sdk/build-tools/36.0.0"
platform_jar="$android_sdk/platforms/android-36/android.jar"
mkdir -p "$package_dir/aar" "$package_dir/classes" "$package_dir/generated" "$package_dir/dex" "$package_dir/apk" "$(dirname "$apk_path")"
unzip -q -o android/app/libs/pinballdreams.aar -d "$package_dir/aar"
python3 - "$project_root/android/app/src/main/AndroidManifest.xml" "$package_dir/AndroidManifest.xml" <<'PY'
import sys
from pathlib import Path
text=Path(sys.argv[1]).read_text()
text=text.replace('<manifest ', '<manifest package="com.olivierh.pinballdreams" ', 1)
Path(sys.argv[2]).write_text(text)
PY
"$tools_dir/aapt2" compile --dir android/app/src/main/res -o "$package_dir/resources.zip"
"$tools_dir/aapt2" link -o "$package_dir/resources.apk" -I "$platform_jar" \
    --manifest "$package_dir/AndroidManifest.xml" --java "$package_dir/generated" \
    --min-sdk-version 23 --target-sdk-version 36 --version-code 1 --version-name 0.1.0 \
    "$package_dir/resources.zip"
"$JAVA_HOME/bin/javac" -source 17 -target 17 -classpath "$platform_jar:$package_dir/aar/classes.jar" \
    -d "$package_dir/classes" \
    android/app/src/main/java/com/olivierh/pinballdreams/MainActivity.java \
    "$package_dir/generated/com/olivierh/pinballdreams/R.java"
"$JAVA_HOME/bin/jar" --create --file "$package_dir/activity.jar" -C "$package_dir/classes" .
"$tools_dir/d8" --lib "$platform_jar" --min-api 23 --output "$package_dir/dex" \
    "$package_dir/aar/classes.jar" "$package_dir/activity.jar"
unzip -q -o "$package_dir/resources.apk" -d "$package_dir/apk"
cp "$package_dir/dex/classes.dex" "$package_dir/apk/classes.dex"
mkdir -p "$package_dir/apk/lib"
cp -R "$package_dir/aar/jni/." "$package_dir/apk/lib/"
raw_apk="$package_dir/unsigned.apk"
rm -f "$raw_apk"
(
    cd "$package_dir/apk"
    zip -q -0 -r "$raw_apk" .
)
"$tools_dir/zipalign" -f -P 16 4 "$raw_apk" "$package_dir/aligned.apk"
"$tools_dir/apksigner" sign --ks "$android_cache/debug.keystore" --ks-key-alias androiddebugkey \
    --ks-pass pass:android --key-pass pass:android --out "$apk_path" "$package_dir/aligned.apk"
"$tools_dir/apksigner" verify "$apk_path"
"$tools_dir/zipalign" -c -P 16 4 "$apk_path"
printf 'APK ready: %s\n' "$apk_path"
