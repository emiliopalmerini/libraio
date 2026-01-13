{ pkgs }:

pkgs.buildGoModule {
  pname = "libraio";
  version = "0.1.0";
  src = pkgs.lib.cleanSource ../.;

  vendorHash = "sha256-Y8e9L1O44A+kmICBSKroHV4XNRY/4+vyaI2sGnFSGjE=";

  subPackages = [ "cmd/libraio" ];

  meta = with pkgs.lib; {
    description = "TUI for managing Obsidian vaults with Johnny Decimal";
    homepage = "https://github.com/emiliopalmerini/libraio";
    license = licenses.mit;
    maintainers = [ ];
  };
}
