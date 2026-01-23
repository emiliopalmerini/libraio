{ pkgs }:

pkgs.buildGoModule {
  pname = "libraio";
  version = "0.2.0";
  src = pkgs.lib.cleanSource ../.;

  vendorHash = "sha256-FKeVWxOLNvGcN7X9Q9ALLB9b5Ns0vMbXrt2T6Ww0M0A=";

  subPackages = [ "cmd/libraio" "cmd/libraio-cli" ];

  meta = with pkgs.lib; {
    description = "TUI and CLI for managing Obsidian vaults with Johnny Decimal";
    homepage = "https://github.com/emiliopalmerini/libraio";
    license = licenses.mit;
    maintainers = [ ];
  };
}
