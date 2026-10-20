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

  enterShell = ''
    export PYTHONPATH="${config.git.root}/src/fraud-service/gen:$PYTHONPATH"
  '';

  tasks."codegen:proto" = {
    exec = ''
      set -euo pipefail
      cd "${config.git.root}"
      mkdir -p src/fraud-service/gen src/simulator/gen
      while IFS= read -r -d $'\0' proto; do
        for output in src/fraud-service/gen src/simulator/gen; do
          protoc \
            -I contracts \
            --python_out="$output" --pyi_out="$output" \
            --plugin=protoc-gen-grpc_python=${pkgs.grpc}/bin/grpc_python_plugin \
            --grpc_python_out="$output" \
            "$proto"
        done
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
          "src/fraud-service/gen/**"
          "src/simulator/gen/**"
        ];
      };
      gofumpt = {
        enable = true;
        extra = true;
      };
    };
  };
}
