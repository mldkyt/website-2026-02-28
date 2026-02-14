{ pkgs, lib, config, inputs, ... }:

{
  # https://devenv.sh/packages/
  packages = [ pkgs.watchexec ];

  # https://devenv.sh/languages/
  # languages.rust.enable = true;
  languages.go.enable = true;
  languages.javascript = {
    enable = true;
    directory = "${config.devenv.root}/web";
    pnpm = {
      enable = true;
      install.enable = true;
    };
  };

  dotenv.enable = true;

  processes.dev.exec = "${lib.getExe pkgs.watchexec} -e go,astro,css -- bash ./dev.sh";

  # https://devenv.sh/git-hooks/
  # git-hooks.hooks.shellcheck.enable = true;

  # See full reference at https://devenv.sh/reference/options/
}
