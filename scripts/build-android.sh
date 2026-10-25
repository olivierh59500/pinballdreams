#!/bin/sh
# Build the conventional Gradle APK around the Go/Ebitengine binding.
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$script_dir/android-env.sh"
cd "$project_root"
mkdir -p android/app/libs "$android_cache/bind-tools" "$ANDROID_USER_HOME"
ebiten_version=$(go list -m -f '{{.Version}}' github.com/hajimehoshi/ebiten/v2)
binding_tools="$android_cache/bind-tools"
cat > "$binding_tools/go.mod" <<EOF
module pinballdreams-android-tools

go 1.26.0

require github.com/hajimehoshi/ebiten/v2 $ebiten_version
EOF
(
    cd "$binding_tools"
    CGO_ENABLED=0 go build -mod=mod -o "$binding_tools/ebitenmobile" github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile
)
CGO_ENABLED=0 \
CGO_LDFLAGS="${CGO_LDFLAGS:-} -Wl,-z,max-page-size=16384 -Wl,-z,common-page-size=16384" \
"$binding_tools/ebitenmobile" bind -target "${PINBALL_ANDROID_TARGET:-android/arm64}" -androidapi 23 \
    -javapkg com.olivierh.pinballdreams -o android/app/libs/pinballdreams.aar ./mobile
if [ ! -f "$android_cache/debug.keystore" ]; then
    "$JAVA_HOME/bin/keytool" -genkeypair -keystore "$android_cache/debug.keystore" -storetype PKCS12 \
        -storepass android -alias androiddebugkey -keypass android -dname "CN=Android Debug,O=Android,C=US" \
        -keyalg RSA -keysize 2048 -validity 10000
fi
if [ "${PINBALL_ANDROID_PACKAGE_MODE:-gradle}" = direct ]; then
    exec "$script_dir/package-android.sh"
fi
./android/gradlew -p android --console=plain :app:assembleDebug
"$android_sdk/build-tools/36.0.0/apksigner" verify "$apk_path"
"$android_sdk/build-tools/36.0.0/zipalign" -c -P 16 4 "$apk_path"
printf 'APK ready: %s\n' "$apk_path"
