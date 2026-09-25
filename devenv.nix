{ pkgs, config, ... }:
{
  packages = with pkgs; [
    kubernetes-helm
    kubectl

    ruff
    gofumpt
    golangci-lint

    grpc
    protobuf
    protoc-gen-go
    protoc-gen-go-grpc

    openspec
    zlib
  ];

  languages = {
    go = {
      delve = {
        enable = true;
      };
      enable = true;
      lsp = {
        enable = true;
      };
    };
    python = {
      directory = "src/fraud-service";
      enable = true;
      lsp = {
        enable = true;
      };
      uv = {
        enable = true;
        sync = {
          enable = true;
        };
      };
      venv = {
        enable = false;
      };
      version = "3.14";
    };
  };

  tasks."codegen:proto" = {
    exec = ''
      set -euo pipefail
      cd "${config.git.root}"
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

  git-hooks.hooks = {
    treefmt.enable = true;
  };

  treefmt = {
    enable = true;
    config.settings.global.excludes = [
      "infra/**/templates/**"
      "infra/**/charts/*.tgz"
    ];
    config.programs = {
      ruff-format = {
        enable = true;
        excludes = [
          "src/**/fraud/v1/**"
        ];
      };
      gofumpt = {
        enable = true;
        extra = true;
        excludes = [
          "src/**/gen/fraud/v1/**"
        ];
      };
    };
  };
}
