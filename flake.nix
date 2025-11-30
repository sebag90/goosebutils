{
  description = "Goosebutils — multiple Go utilities built with Nix flakes";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";  
  };

  outputs = { self, nixpkgs, ... }:
    let
      system = builtins.currentSystem;
      pkgs = import nixpkgs { inherit system; };
    in {

      packages.${system} = {
        search = pkgs.buildGoModule {
          pname = "search";
          version = "0.1.0";
          src = goosebutils;
          subPackages = [ "search" ];
          vendorHash = null;  # Let Nix compute it for you (use `nix build` to get the hash)
        };

        # Build the `replace` binary
        replace = pkgs.buildGoModule {
          pname = "replace";
          version = "0.1.0";
          src = goosebutils;
          subPackages = [ "replace" ];
          vendorHash = null;
        };

        # Build the `dstroy` binary
        dstroy = pkgs.buildGoModule {
          pname = "dstroy";
          version = "0.1.0";
          src = goosebutils;
          subPackages = [ "dstroy" ];
          vendorHash = null;
        };
         
        devenver = pkgs.buildGoModule {
          pname = "devenver";
          version = "0.1.0";
          src = goosebutils;
          subPackages = [ "devenver" ];
          vendorHash = null;
        };

        # Meta-package containing all three
        default = pkgs.buildEnv {
          name = "goosebutils";
          paths = [
            self.packages.${system}.search
            self.packages.${system}.replace
            self.packages.${system}.dstroy
            self.packages.${system}.devenver
          ];
        };
      };

      defaultPackage.${system} = self.packages.${system}.default;
    };
}

