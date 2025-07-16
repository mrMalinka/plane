{ pkgs ? import <nixpkgs> {} }:
pkgs.mkShell rec {
    buildInputs = with pkgs; [
      inotify-tools
      nodejs
      android-studio
      picocom

      # tinygo doesnt support 1.24
      go_1_23
      tinygo
    ];

    shellHook = builtins.readFile ./shell-hook.bash;
}
