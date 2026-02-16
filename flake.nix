{
  description = "Anytype Bundle: Prepackaged All-in-One Self-Hosting";

  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    inputs@{
      self,
      flake-parts,
      nixpkgs,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [ ];
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];

      perSystem =
        {
          config,
          self',
          inputs',
          pkgs,
          system,
          lib,
          ...
        }:
        let
          # Version information - extract from git or use defaults
          version = if self ? rev then self.shortRev else "dev";
          commit = self.rev or "dirty";
          date = self.lastModifiedDate or "1970-01-01T00:00:00Z";

          # Format the date for build info
          buildDate = builtins.substring 0 10 date;
        in
        {
          packages.default = (pkgs.buildGoModule.override { go = pkgs.go_1_25; }) rec {
            pname = "any-sync-bundle";
            version = "v1.3.0-2026-01-31";

            src = ./.;

            vendorHash = "sha256-0AnVTgSu9/wGmww+la92yC4wDdPq1LTs21ej1Nz2+UI=";

            env.CGO_ENABLED = 0;

            ldflags = [
              "-w"
              "-s"
              "-X github.com/grishy/any-sync-bundle/cmd.version=${version}"
              "-X github.com/grishy/any-sync-bundle/cmd.commit=${commit}"
              "-X github.com/grishy/any-sync-bundle/cmd.date=${buildDate}"
            ];

            doCheck = true;

            meta = with lib; {
              description = "Anytype Bundle: All-in-one self-hosted Anytype server";
              homepage = "https://github.com/grishy/any-sync-bundle";
              license = licenses.mit;
              maintainers = [ ];
              platforms = platforms.unix;
              mainProgram = "any-sync-bundle";
            };
          };

          # Development shell with all necessary tools
          devShells.default = pkgs.mkShell {
            name = "any-sync-bundle-dev";

            packages = with pkgs; [
              git
              go_1_25

              # Development tools
              golangci-lint
              goreleaser

              # Container tools (optional, for testing)
              docker
              docker-compose
            ];

            shellHook = ''
              echo "🚀 any-sync-bundle development environment"
              echo ""
              echo "Available commands:"
              echo "  go build          - Build the binary"
              echo "  go test ./...     - Run tests"
              echo "  golangci-lint run - Run linter"
              echo "  goreleaser build  - Build with goreleaser"
              echo "  nix build         - Build with Nix"
              echo ""
              echo "Go version: $(go version)"
            '';
          };

          # Format check
          formatter = pkgs.nixpkgs-fmt;
        };

      flake = { };
    };
}
