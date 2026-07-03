{
  description = "Mirror GitHub repositories to Radicle";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    treefmt-nix.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs =
    {
      self,
      nixpkgs,
      treefmt-nix,
    }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      treefmtEval = forAllSystems (pkgs: treefmt-nix.lib.evalModule pkgs ./nix/treefmt.nix);
    in
    {
      nixosModules.default = ./nix/module.nix;

      packages = forAllSystems (pkgs: rec {
        default = pkgs.callPackage ./nix/package.nix { };
        radicle-mirror = default;
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.callPackage ./nix/devshell.nix { };
      });

      formatter = forAllSystems (pkgs: treefmtEval.${pkgs.system}.config.build.wrapper);

      # export every package and devshell as a check so CI builds them all
      checks = forAllSystems (
        pkgs:
        let
          prefixNames = prefix: nixpkgs.lib.mapAttrs' (n: v: nixpkgs.lib.nameValuePair "${prefix}-${n}" v);
        in
        (prefixNames "package" self.packages.${pkgs.system})
        // (prefixNames "devShell" self.devShells.${pkgs.system})
        // {
          formatting = treefmtEval.${pkgs.system}.config.build.check self;
        }
      );
    };
}
