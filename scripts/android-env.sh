#!/bin/sh
# Keep all disposable Android caches and outputs within this checkout.
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
android_cache="$project_root/.cache/android"
android_sdk=${ANDROID_HOME:-${ANDROID_SDK_ROOT:-/opt/homebrew/share/android-commandlinetools}}
java_home_path=${JAVA_HOME:-/opt/homebrew/opt/openjdk@21}
android_ndk=${ANDROID_NDK_HOME:-"$android_sdk/ndk/28.2.13676358"}
if [ ! -f "$android_sdk/platforms/android-36/android.jar" ]; then
    android_sdk="$HOME/Library/Android/sdk"
fi
if [ ! -f "$android_sdk/platforms/android-36/android.jar" ] || [ ! -f "$android_ndk/meta/platforms.json" ]; then
    echo "Android SDK 36 and NDK 28.2.13676358 are required; set ANDROID_HOME and ANDROID_NDK_HOME." >&2
    exit 1
fi
if [ ! -x "$java_home_path/bin/java" ]; then
    java_home_path="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
fi
export JAVA_HOME="$java_home_path"
export ANDROID_HOME="$android_sdk"
export ANDROID_SDK_ROOT="$android_sdk"
export ANDROID_NDK_HOME="$android_ndk"
export ANDROID_USER_HOME="$android_cache/sdk-user"
export PATH="$java_home_path/bin:$android_sdk/platform-tools:$PATH"
export GOPATH="$android_cache/go"
export GOMODCACHE="$GOPATH/pkg/mod"
export GOCACHE="$project_root/.cache/go-build"
export GRADLE_USER_HOME="$android_cache/gradle"
export GOWORK=off
export GOTOOLCHAIN=local
apk_path="$project_root/android/app/build/outputs/apk/debug/app-debug.apk"
