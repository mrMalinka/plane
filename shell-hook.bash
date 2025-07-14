# load .env
export $(grep -v '^#' .env | xargs)
# variables needed:
# PROJECT_ROOT - absolute path to the repo
# ZERO_USERNAME - username of the user on the pi zero
# ZERO_IP - ip address of the pi zero (or hostname)
# ZERO_SYNC_DIR - absolute path to the directory on the pi zero to sync (make sure it exists)

# pico
find_pico_device() {
    PICO_DEVICE=""
    for dev in /dev/sd?1; do
        if [ -b "$dev" ]; then
        PICO_DEVICE="$dev"
        break
        fi
    done

    if [ -z "$PICO_DEVICE" ]; then
        echo "pico is not connected" >&2
        return 1
    else
        echo "$PICO_DEVICE"
    fi
}
pico_mount() {
    DEV="$(find_pico_device)"
    if [ -z "$DEV" ]; then
        return 1
    fi
    mkdir -p "$PROJECT_ROOT/pico/mount/"
    sudo mount "$DEV" "$PROJECT_ROOT/pico/mount/" -o uid=1000,gid=1000,flush
}
pico_sync() {
    # check if pico is mounted and has python files before deleting the local ones
    if ! find "$PROJECT_ROOT/pico/mount/" -maxdepth 1 -name '*.py' | grep -q .; then
        echo "pico is not mounted" >&2
        return 1
    fi

    mkdir -p "$PROJECT_ROOT/pico/save/"
    rm -f "$PROJECT_ROOT/pico/save/"*.py
    cp "$PROJECT_ROOT/pico/mount/"*.py "$PROJECT_ROOT/pico/save/"

    rm -f "$PROJECT_ROOT/pico/libs.txt"
    {
        echo "from 'https://circuitpython.org/libraries' first bundle"
        ls -1 "$PROJECT_ROOT/pico/mount/lib/"
    } > "$PROJECT_ROOT/pico/libs.txt"
}
alias pico_unmount='sudo umount "$PROJECT_ROOT/pico/mount/"'
alias pico_term='sudo picocom -b 115200 /dev/ttyACM0'

# tailwind
APP_ASSETS="$PROJECT_ROOT/mobile/src/app/src/main/assets"
tailwind_update() {
    BEFORE=$PWD
    cd "$APP_ASSETS"
    npx --silent @tailwindcss/cli -i "styles.css" -o "tailwind.css"
    cd "$BEFORE"
}
while inotifywait -qq -e modify "$APP_ASSETS"; do
    tailwind_update >/dev/null 2>&1
done &

# pi zero
zero_sync() {
    rsync -avz -e ssh "$PROJECT_ROOT"/zero/ "$ZERO_USERNAME"@"$ZERO_IP":"$ZERO_SYNC_DIR"/
}
alias zero_ssh="ssh $ZERO_USERNAME@$ZERO_IP"

# android studio
alias android_studio_launch='android-studio >/dev/null 2>&1 &'

# rust stuff
export PATH=$PATH:''${CARGO_HOME:-~/.cargo}/bin
export PATH=$PATH:''${RUSTUP_HOME:-~/.rustup}/toolchains/$RUSTC_VERSION-x86_64-unknown-linux-gnu/bin/

code .