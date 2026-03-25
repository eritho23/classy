{
  config,
  pkgs,
  modulesPath,
  ...
}: {
  imports = [
    "${modulesPath}/profiles/minimal.nix"
    "${modulesPath}/profiles/perlless.nix"
    "${modulesPath}/profiles/qemu-guest.nix"
    "${modulesPath}/virtualisation/qemu-vm.nix"
  ];

  virtualisation = {
    graphics = false;
    forwardPorts = [
      {
        from = "host";
        host.port = 8080;
        guest.port = 8080;
      }
    ];
  };

  systemd.services.postgresql-setpassword = {
    requires = [
      config.systemd.services.postgresql.name
      "postgresql-setup.service"
    ];
    after = [
      config.systemd.services.postgresql.name
      "postgresql-setup.service"
    ];

    path = [pkgs.postgresql];

    serviceConfig = {
      User = "postgres";
    };

    script = ''
      psql -U postgres -c "alter role classy with password '12345678';"
    '';
  };

  services = {
    getty.autologinUser = "root";

    postgresql = {
      enable = true;
      ensureUsers = [
        {
          name = "classy";
          ensureDBOwnership = true;
          ensureClauses.login = true;
        }
      ];
      ensureDatabases = ["classy"];
    };

    classy = {
      enable = true;

      httpOrigin = "http://127.0.0.1:8080";

      databaseUrlPath = pkgs.writeText "connection-string" "postgres://classy:12345678@/classy?host=/run/postgresql";
    };

    nginx = {
      enable = true;
      virtualHosts = {
        "classy" = {
          listen = [
            {
              addr = "0.0.0.0";
              port = 8080;
            }
          ];
          locations."/" = {
            proxyPass = "http://unix:/run/classy/http.sock";
          };
        };
      };
    };
  };

  system.stateVersion = "25.11";
}
