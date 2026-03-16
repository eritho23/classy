_: {
  projectRootFile = "flake.nix";

  programs = {
    alejandra.enable = true;
    deadnix.enable = true;
    gofumpt.enable = true;
    mbake.enable = true;
    sqlfluff = {
      enable = true;
      dialect = "postgres";
    };
    statix.enable = true;
    templ.enable = true;
    yamlfmt.enable = true;
  };
}
