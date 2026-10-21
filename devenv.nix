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
      enable = true;
      lsp = {
        enable = true;
      };
      uv = {
        enable = true;
      };
      version = "3.14";
    };
  };

  tasks."codegen:proto" = {
    exec = ''bash "${config.git.root}/scripts/codegen-proto.sh" "${pkgs.grpc}/bin/grpc_python_plugin"'';
    before = [ "devenv:enterShell" ];
    execIfModified = [
      "contracts/**/*.proto"
      "scripts/codegen-proto.sh"
      "devenv.lock"
      "devenv.nix"
    ];
  };

  git-hooks.hooks = {
    treefmt.enable = true;
  };

  treefmt = {
    enable = true;
    config.programs = {
      ruff-format = {
        enable = true;
        excludes = [
          "src/fraud-service/gen/**"
        ];
      };
      gofumpt = {
        enable = true;
        extra = true;
      };
    };
  };
}
