{ pkgs }:

pkgs.buildGoModule {
  pname = "libraio";
  version = "0.2.0";
  src = pkgs.lib.cleanSource ../.;

  vendorHash = "sha256-U8QGadwNdZRMjaTkt37l3H9X1Deqr4QH2tvblQwg1mw=";

  subPackages = [ "cmd/libraio" "cmd/libraio-cli" "cmd/libraio-mcp" ];

  meta = with pkgs.lib; {
    description = "TUI and CLI for managing Obsidian vaults with Johnny Decimal";
    homepage = "https://github.com/emiliopalmerini/libraio";
    license = licenses.mit;
    maintainers = [ ];
  };
}
