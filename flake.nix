{
  description = "Libraio - TUI and CLI for managing Obsidian vaults with Johnny Decimal";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          libraio = pkgs.callPackage ./nix/package.nix {};
          libraio-cli = pkgs.callPackage ./nix/package-cli.nix {};
          default = self.packages.${system}.libraio;
        };

        devShells.default = pkgs.callPackage ./nix/devShell.nix {};
      }
    );
}
