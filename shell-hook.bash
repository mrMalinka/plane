# load .env
export $(grep -v '^#' .env | xargs)
# variables needed:
# PROJECT_ROOT - absolute path to the repo
# ZERO_USERNAME - username of the user on the pi zero
# ZERO_IP - ip address of the pi zero (or hostname)
# ZERO_SYNC_DIR - absolute path to the directory on the pi zero to sync (make sure it exists)

# pico
alias pico_flash='tinygo flash -target=pico' # followed by the target
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

# pico vendor setup so gopls works
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
    printf "module %s\ngo 1.24.4\n" "$module_name" > "$dir/go.mod"
  done
done

# launch vsc
cd "$PROJECT_ROOT"
code .