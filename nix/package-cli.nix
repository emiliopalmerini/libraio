{ pkgs }:

pkgs.buildGoModule {
  pname = "libraio-cli";
  version = "0.2.0";
  src = pkgs.lib.cleanSource ../.;

  vendorHash = "sha256-Y8e9L1O44A+kmICBSKroHV4XNRY/4+vyaI2sGnFSGjE=";

  subPackages = [ "cmd/libraio-cli" ];

  meta = with pkgs.lib; {
    description = "CLI for managing Obsidian vaults with Johnny Decimal";
    homepage = "https://github.com/emiliopalmerini/libraio";
    license = licenses.mit;
    maintainers = [ ];
  };
}
