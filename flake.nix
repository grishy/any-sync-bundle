{
  description = "Anytype Bundle: Prepackaged All-in-One Self-Hosting";

  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [ ];
      systems = [ "x86_64-linux" "aarch64-linux" "aarch64-darwin" "x86_64-darwin" ];
      perSystem = { config, self', inputs', pkgs, system, ... }: {
        packages.default = pkgs.buildGoModule rec {
          pname = "any-sync-bundle";
          version = "v0.5.0+2024-12-18";

          src = ./.;

          vendorHash = "sha256-rSNmurbehNPYQBgkrPjI+4W/8pTT4zJlI4EuGCNlBzQ=";

          env.CGO_ENABLED = 0;
          ldflags = [
            "-w -s"
            "-X github.com/grishy/any-sync-bundle/cmd.version=${version}"
          ];
        };
      };
      flake = { };
    };
}
