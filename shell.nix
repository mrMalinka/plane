{ pkgs ? import <nixpkgs> {} }:
pkgs.mkShell rec {
    buildInputs = with pkgs; [
      inotify-tools
      nodejs
      android-studio
      picocom
    ];

    shellHook = builtins.readFile ./shell-hook.bash;
}
