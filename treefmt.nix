{
  lib,
  pkgs,
  ...
}: {
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

  settings.formatter."sql-formatter-custom" = {
    command = lib.getExe pkgs.bash;
    options = let
      sqlFormatterOptions = {
        paramTypes.named = ["@"];
      };
      sqlFormatterOptionsJsonString = builtins.toJSON sqlFormatterOptions;
    in [
      "-euc"
      ''
        for file in "$@"; do
          ${lib.getExe pkgs.sql-formatter} --config '${sqlFormatterOptionsJsonString}' -l postgresql --fix "$file"
        done
      ''
      "--"
    ];
    includes = ["*.sql"];
  };
}
