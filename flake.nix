{
  description = "nixs — unified Nix search CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls          # LSP for neovim
            gotools        # goimports, godoc, etc
            go-tools       # staticcheck
            delve          # debugger
          ];

          shellHook = ''
            echo "nixs dev shell"
            echo "go $(go version | awk '{print $3}')"
          '';
        };

        # so you can also do: nix build
        packages.default = pkgs.buildGoModule {
          pname = "nixs";
          version = "0.1.0";
          src = ./.;
          vendorHash = null; 
        };
      });
}
