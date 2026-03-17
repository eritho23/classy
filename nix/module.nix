{
  lib,
  config,
  pkgs,
  ...
}: let
  cfg = config.services.classy;
  inherit (lib) mkEnableOption mkOption mkIf types getExe;
  classyPkg = pkgs.callPackage ../default.nix {};
in {
  options.services.classy = {
    enable = mkEnableOption "The classy service.";

    socketPath = mkOption {
      default = "%t/http.sock";
      type = types.oneOf [
        types.path
        types.str
      ];
      description = "The socket path where the application will listen.";
    };

    databaseUrlPath = mkOption {
      type = types.path;
      description = "The path of a file containing the database URL.";
    };
  };

  config = mkIf cfg.enable {
    systemd.services.classy = {
      serviceConfig = {
        ExecStart = getExe classyPkg;
        DynamicUser = true;
        LoadCredential = [
          "database_url:${cfg.databaseUrlPath}"
        ];
        Environment = [
          "HTTP_SOCKET_PATH=${cfg.socketPath}"
        ];
      };
    };
  };
}
