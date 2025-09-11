{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:

{

  languages.javascript = {
    enable = true;
    pnpm.enable = true;
    pnpm.install.enable = true;
  };

  processes."dev-server" = {
    exec = ''pnpm run dev'';
  };

}
