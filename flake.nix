{
  description = "Open CI Control Plane — Go development environment";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "aarch64-darwin"
        "x86_64-darwin"
        "aarch64-linux"
        "x86_64-linux"
      ];

      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      documentedOpaVersion = "1.21.2";
    in
    {
      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          name = "open-ci-control-plane";

          packages = [
            # OPA v1.21.2 declares `go 1.26.0`, making 1.26 the floor.
            pkgs.go_1_26
            pkgs.gopls
            pkgs.gotools
            pkgs.delve

            pkgs.golangci-lint
            pkgs.go-tools
            pkgs.govulncheck

            pkgs.open-policy-agent
            pkgs.regal

            pkgs.syft
            pkgs.cosign

            pkgs.jq
          ];

          env = {
            GOFLAGS = "-mod=readonly";
            CGO_ENABLED = "0";
          };

          shellHook = ''
            echo "open-ci-control-plane dev shell"
            echo "  go         $(go version | cut -d' ' -f3)"
            echo "  opa        $(opa version | awk '/^Version:/ {print $2}')"
            echo "  regal      $(regal version 2>/dev/null | awk '/^Version:/ {print $2}')"
            echo "  golangci   $(golangci-lint --version | awk '{print $4}')"

            opa_have=$(opa version | awk '/^Version:/ {print $2}')
            if [ "$opa_have" != "${documentedOpaVersion}" ]; then
              echo ""
              echo "  note: opa CLI is $opa_have; knowledge-base/technology/ documents ${documentedOpaVersion}."
              echo "        Version-dependent findings there may not hold. The embedded engine"
              echo "        version comes from go.mod and is unaffected."
            fi
          '';
        };
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt);
    };
}
