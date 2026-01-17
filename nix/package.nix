{ pkgs }:

pkgs.buildGoModule {
  pname = "libraio";
  version = "0.2.0";
  src = pkgs.lib.cleanSource ../.;

  vendorHash = "sha256-6/WETwkyqYlE5lThsC3a+jO6bnqmnyjCN4l71vSUXak=";

  subPackages = [ "cmd/libraio" "cmd/libraio-cli" ];

  meta = with pkgs.lib; {
    description = "TUI and CLI for managing Obsidian vaults with Johnny Decimal";
    homepage = "https://github.com/emiliopalmerini/libraio";
    license = licenses.mit;
    maintainers = [ ];
  };
}
