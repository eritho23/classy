_: {
  projectRootFile = "flake.nix";

  programs = {
    alejandra.enable = true;
    deadnix.enable = true;
    gofumpt.enable = true;
    mbake.enable = true;
    sql-formatter = {
      enable = true;
      dialect = "postgresql";
    };
    statix.enable = true;
    templ.enable = true;
    yamlfmt.enable = true;
  };
}
