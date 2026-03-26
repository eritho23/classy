_: {
  lib,
  config,
  pkgs,
  ...
}: let
  cfg = config.services.classy;
  inherit (lib) mkEnableOption mkOption mkIf types getExe;
  classyPkg = pkgs.callPackage ../default.nix {};
  goMigratePkg = pkgs.callPackage ./go-migrate.nix {};
  systemdServices = config.systemd.services;
in {
  options.services.classy = {
    enable = mkEnableOption "The classy service.";

    socketPath = mkOption {
      default = "%t/classy/http.sock";
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

    httpOrigin = mkOption {
      type = types.str;
      description = "The HTTP origin where the app is served, used for CSRF protection.";
    };
  };

  config = mkIf cfg.enable {
    systemd.services.classy = let
      servicePrerequisites =
        if (builtins.hasAttr "postgresql-setup" systemdServices)
        then [systemdServices.postgresql-setup.name]
        else [];
    in {
      after = [systemdServices.postgresql.name] ++ servicePrerequisites;
      requires = [systemdServices.postgresql.name] ++ servicePrerequisites;
      wantedBy = ["multi-user.target"];
      environment = {
        HTTP_SOCKET_PATH = cfg.socketPath;
        ORIGIN = cfg.httpOrigin;
      };
      serviceConfig = {
        ExecStartPre = pkgs.writeShellScript "classy-exec-start-pre-migrate-up" ''
          ${lib.getExe goMigratePkg} -path ${../migrations} -database "$(cat $CREDENTIALS_DIRECTORY/database_url)" up
        '';
        ExecStart = getExe classyPkg;
        DynamicUser = true;
        LoadCredential = [
          "database_url:${cfg.databaseUrlPath}"
        ];
        SystemCallFilter = "@system-service";
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
        ];
        NoNewPrivileges = true;
        RuntimeDirectory = "classy";
        StateDirectory = "classy";
        WorkingDirectory = "%S/classy";
        UMask = "007";
        Group = "classy";
      };
    };
  };
}
