{ pkgs, ... }:
{
  packages = [
    pkgs.kubernetes-helm
    pkgs.kubectl
    pkgs.openspec
    pkgs.ruff
    pkgs.zlib
  ];
  languages.go = {
    enable = true;
    delve.enable = true;
    lsp.enable = true;
  };
  languages.python = {
    enable = true;
    version = "3.14";
    directory = "src/fraud-service";
    venv.enable = false;
    uv = {
      enable = true;
      sync.enable = true;
    };
    lsp = {
      enable = true;
    };
  };
  treefmt = {
    enable = true;
    config.settings.global.excludes = [
      "infra/**/templates/**"
      "infra/**/charts/*.tgz"
      "src/fraud-service/fraud/**"
    ];
    config.programs = {
      ruff-format = {
        enable = true;
      };
    };
  };
}
