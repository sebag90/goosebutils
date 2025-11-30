{
  description = "Goosebutils — multiple Go utilities";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
  };

  outputs = inputs@{ self, nixpkgs, ... }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: builtins.listToAttrs (map (system: {
        name = system;
        value = f system;
      }) supportedSystems);
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          search = pkgs.buildGoModule {
            pname = "search";
            version = "0.1.0";
            src = self;
            subPackages = [ "search" ];
            vendorHash = null;
          };

          # replace = pkgs.buildGoModule {
          #   pname = "replace";
          #   version = "0.1.0";
          #   src = self;
          #   subPackages = [ "replace" ];
          #   vendorHash = null;
          # };

          dstroy = pkgs.buildGoModule {
            pname = "dstroy";
            version = "0.1.0";
            src = self;
            subPackages = [ "dstroy" ];
            vendorHash = null;
          };

          devenver = pkgs.buildGoModule {
            pname = "devenver";
            version = "0.1.0";
            src = self;
            subPackages = [ "devenver" ];
            vendorHash = null;
          };

          default = pkgs.buildEnv {
            name = "goosebutils";
            paths = [
              self.packages.${system}.search
              # self.packages.${system}.replace
              self.packages.${system}.dstroy
              self.packages.${system}.devenver
            ];
          };
        });

      # So `nix build` works without specifying a system
      defaultPackage = forAllSystems (system: self.packages.${system}.default);
    };
}
