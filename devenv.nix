{ pkgs, config, ... }:
{
  packages = [
    pkgs.kubernetes-helm
    pkgs.kubectl
    pkgs.openspec
    pkgs.ruff
    pkgs.protobuf
    pkgs.grpc
    pkgs.protoc-gen-go
    pkgs.protoc-gen-go-grpc
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
  tasks."codegen:proto" = {
    exec = ''
      set -euo pipefail
      cd "${config.git.root}"
      mkdir -p src/edge/gen
      while IFS= read -r -d $'\0' proto; do
        protoc \
          -I contracts \
          --python_out=src/fraud-service --pyi_out=src/fraud-service \
          --plugin=protoc-gen-grpc_python=${pkgs.grpc}/bin/grpc_python_plugin \
          --grpc_python_out=src/fraud-service \
          --go_out=src/edge/gen --go_opt=paths=source_relative \
          --go-grpc_out=src/edge/gen --go-grpc_opt=paths=source_relative \
          "$proto"
      done < <(find contracts -type f -name '*.proto' -print0)
    '';
    before = [ "devenv:enterShell" ];
    execIfModified = [
      "contracts/**/*.proto"
      "devenv.lock"
      "devenv.nix"
    ];
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
