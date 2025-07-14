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

      inotify-tools
      nodejs
      android-studio
      picocom
    ];

    shellHook = builtins.readFile ./shell-hook.bash;

    RUSTC_VERSION = overrides.toolchain.channel;
    LIBCLANG_PATH = pkgs.lib.makeLibraryPath [ pkgs.llvmPackages_latest.libclang.lib ];
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
