# load .env
export $(grep -v '^#' .env | xargs)
# variables needed:
# PROJECT_ROOT - absolute path to the repo
# ZERO_USERNAME - username of the user on the pi zero
# ZERO_IP - ip address of the pi zero (or hostname)
# ZERO_SYNC_DIR - absolute path to the directory on the pi zero to sync (make sure it exists)

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
    rsync --exclude="go.mod" \
    -avz -e ssh "$PROJECT_ROOT"/zero/ \
    "$ZERO_USERNAME"@"$ZERO_IP":"$ZERO_SYNC_DIR"/
}
alias zero_ssh="ssh $ZERO_USERNAME@$ZERO_IP"

# android studio
alias android_studio_launch='android-studio >/dev/null 2>&1 &'

# rust stuff
export PATH=$PATH:''${CARGO_HOME:-~/.cargo}/bin
export PATH=$PATH:''${RUSTUP_HOME:-~/.rustup}/toolchains/$RUSTC_VERSION-x86_64-unknown-linux-gnu/bin/

# make gopls work
MODULES="$PROJECT_ROOT/pico/modules"
TINYGOROOT="$(tinygo env TINYGOROOT)"

rm -rf "$MODULES"
rm -rf "$PROJECT_ROOT/pico/vendor"
mkdir -p "$MODULES"

for pkg in machine device runtime; do
  rsync -a --no-perms --no-owner --no-group \
    "$TINYGOROOT"/src/"$pkg" "$MODULES"
  chmod -R u+w "$MODULES"/"$pkg"

  base_dir="${MODULES%/}"
  find "$MODULES/$pkg" -type d -print0 | while IFS= read -r -d '' dir; do
    module_name="${dir#$base_dir/}"
    printf "module %s\ngo 1.23.8\n" "$module_name" > "$dir/go.mod"
  done
done

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
pico_flash() {
    if ! mountpoint -q "$PROJECT_ROOT"/pico/mount; then
        echo "pico is not mounted, mounting"
        pico_mount
    fi

    tinygo build -o main.uf2 -target=pico "$PROJECT_ROOT"/pico/
    sudo mv main.uf2 "$PROJECT_ROOT"/pico/mount/
}
alias pico_unmount='sudo umount "$PROJECT_ROOT/pico/mount/"'
alias pico_term='sudo picocom -b 115200 /dev/ttyACM0'

mod_build_mode_on() {
    # comments out the necessary lines from go.mod (breaks gopls but fixes tinygo)
    awk '
    BEGIN { in_replace = 0 }
    /^[[:space:]]*replace \(/ {
        in_replace = 1
        print
        next
    }
    in_replace {
        if (/^[[:space:]]*\)/) {
            in_replace = 0
            print
        } else if (!/^[[:space:]]*\//) {
            print "//" $0
        } else {
            print
        }
        next
    }
    /^[[:space:]]*\// || /^[[:space:]]*(go 1\.|module|require \(|\))/ {
        print
        next
    }
    NF == 0 {
        print "//"
        next
    }
    $1 ~ /\./ {
        print
        next
    }
    {
        print "//" $0
    }
    ' go.mod > go.mod.tmp && mv go.mod.tmp go.mod
    go mod vendor
}
mod_build_mode_off() {
    # undoes the previous function (breaks tinygo but fixes gopls)
    awk '
    BEGIN { in_replace = 0 }
    /^[[:space:]]*replace \(/ {
        in_replace = 1
        print
        next
    }
    in_replace {
        if (/^[[:space:]]*\)/) {
            in_replace = 0
        }
        if (/^\/\//) {
            sub(/^\/\//, "")
        }
        print
        next
    }
    /^[[:space:]]*(go 1\.|module|require \(|\))/ {
        print
        next
    }
    /^\/\// {
        original = substr($0, 3)
        if (original !~ /^[[:space:]]*\/\//) {
            split(original, parts, /[[:space:]]+/)
            first_word = parts[1]
            if (first_word !~ /\./) {
                print original
                next
            }
        }
        print
        next
    }
    { print }
    ' go.mod > go.mod.tmp && mv go.mod.tmp go.mod
    go mod vendor
}

# launch vsc
cd "$PROJECT_ROOT"
code .
