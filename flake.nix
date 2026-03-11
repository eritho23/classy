{
  description = "Flake for the classy service.";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = {
    self,
    nixpkgs,
    treefmt-nix,
    ...
  }: let
    linuxSystems = [
      "aarch64-linux"
      "x86_64-linux"
    ];
    eachSystem = f: nixpkgs.lib.genAttrs linuxSystems (system: f nixpkgs.legacyPackages.${system});
    treefmtEval = eachSystem (pkgs: treefmt-nix.lib.evalModule pkgs ./treefmt.nix);
    rev =
      if (self ? rev)
      then self.rev
      else "dirty";
  in {
    formatter = eachSystem (pkgs: treefmtEval.${pkgs.stdenv.hostPlatform.system}.config.build.wrapper);
    checks = eachSystem (pkgs: {
      formatting = treefmtEval.${pkgs.stdenv.hostPlatform.system}.config.build.check self;
    });
    devShells = eachSystem (pkgs: {
      default = pkgs.mkShell {
        packages = with pkgs; [
          air
          go
          golangci-lint
          gopls
          gosec
          go-tools
          helix
          nginx
          nixd
          pdpmake
          (pkgs.callPackage ./nix/go-migrate.nix {})
          postgresql.out
          sqlc
          sqlite
          templ
          uutils-coreutils-noprefix
        ];

        shellHook = ''
          alias make='pdpmake'
        '';
      };
    });
    packages = eachSystem (pkgs: rec {
      classy = pkgs.callPackage ./default.nix {inherit rev;};
      default = classy;
    });
  };
}
