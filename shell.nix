{ pkgs ? import <nixpkgs> {} }:
let
    overrides = (builtins.fromTOML (builtins.readFile ./zero/rust-toolchain.toml));
    libPath = with pkgs; lib.makeLibraryPath [
      # load external libraries here
    ];
in pkgs.mkShell rec {
    buildInputs = with pkgs; [
      clang
      llvmPackages_19.bintools
      rustup

      android-studio
      picocom
    ];

    RUSTC_VERSION = overrides.toolchain.channel;
    LIBCLANG_PATH = pkgs.lib.makeLibraryPath [ pkgs.llvmPackages_latest.libclang.lib ];
    shellHook = ''
      export PROJECTROOT=$PWD

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
        mkdir -p "$PROJECTROOT/pico/mount/"
        sudo mount "$DEV" "$PROJECTROOT/pico/mount/" -o uid=1000,gid=1000,flush
      }
      alias pico_unmount='sudo umount "$PROJECTROOT/pico/mount/"'
      alias pico_term='sudo picocom -b 115200 /dev/ttyACM0'
      pico_code_update() {
        # check if pico is mounted and has python files before deleting the local ones
        if ! find "$PROJECTROOT/pico/mount/" -maxdepth 1 -name '*.py' | grep -q .; then
          echo "pico is not mounted" >&2
          return 1
        fi

        mkdir -p "$PROJECTROOT/pico/save/"
        rm -f "$PROJECTROOT/pico/save/"*.py
        cp "$PROJECTROOT/pico/mount/"*.py "$PROJECTROOT/pico/save/"

        rm -f "$PROJECTROOT/pico/libs.txt"
        {
          echo "from 'https://circuitpython.org/libraries' first bundle"
          ls -1 "$PROJECTROOT/pico/mount/lib/"
        } > "$PROJECTROOT/pico/libs.txt"
      }

      alias android_studio_launch='android-studio >/dev/null 2>&1 & disown'

      export PATH=$PATH:''${CARGO_HOME:-~/.cargo}/bin
      export PATH=$PATH:''${RUSTUP_HOME:-~/.rustup}/toolchains/$RUSTC_VERSION-x86_64-unknown-linux-gnu/bin/

      code .
    '';
    RUSTFLAGS = (builtins.map (a: ''-L ${a}/lib'') [
      # add libraries here
    ]);
    LD_LIBRARY_PATH = libPath;
    BINDGEN_EXTRA_CLANG_ARGS =
    (builtins.map (a: ''-I"${a}/include"'') [
      # add dev libraries here
      pkgs.glibc.dev
    ])
    ++ [
      ''-I"${pkgs.llvmPackages_latest.libclang.lib}/lib/clang/${pkgs.llvmPackages_latest.libclang.version}/include"''
      ''-I"${pkgs.glib.dev}/include/glib-2.0"''
      ''-I${pkgs.glib.out}/lib/glib-2.0/include/''
    ];
  }
