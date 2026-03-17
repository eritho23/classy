_: {
  projectRootFile = "flake.nix";

  programs = {
    alejandra.enable = true;
    beautysh.enable = true;
    deadnix.enable = true;
    gofumpt.enable = true;
    mbake.enable = true;
    shellcheck.enable = true;
    statix.enable = true;
    templ.enable = true;
    yamlfmt.enable = true;
  };
}
