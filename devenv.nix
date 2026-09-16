{ pkgs, ... }:
{
  packages = [
    pkgs.kubernetes-helm
    pkgs.kubectl
    pkgs.openspec
    pkgs.ruff
  ];
  languages.python = {
    enable = true;
    version = "3.14";
    directory = "src/fraud-service";
    venv.enable = false;
    uv = {
      enable = true;
      sync.enable = false;
    };
    lsp = {
      enable = true;
    };
  };
  treefmt = {
    enable = true;
    config.settings.global.excludes = [ "deployments/**" ];
    config.programs = {
      ruff-format = {
        enable = true;
      };
    };
  };
}
