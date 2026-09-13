{
  description = "Sheetopia Sync — self-hosted sync server for Sheetopia";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    inputs @ { self, nixpkgs, flake-utils, ... }:
    let
      overlay = final: prev: {
        sheetopia-admin = final.callPackage ./nix/package.nix { };
      };
    in
    flake-utils.lib.eachSystem [ "x86_64-linux" "aarch64-linux" ] (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ overlay ];
        };
      in
      {
        packages = {
          default = pkgs.sheetopia-admin;
          sheetopia-admin = pkgs.sheetopia-admin;
        };

        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go_1_26 pkgs.gitMinimal ];
        };
      }
    ) // {
      # ── Flake-level outputs (not per-system) ─────────────────────────
      overlays.default = overlay;

      nixosModules.default = { config, lib, pkgs, ... }:
        let
          cfg = config.services.sheetopia-sync;
        in
        {
          options.services.sheetopia-sync = {
            enable = lib.mkEnableOption
              "sheetopia-sync, the self-hosted sync server for Sheetopia";

            package = lib.mkOption {
              type = lib.types.package;
              default = pkgs.sheetopia-admin.unwrapped;
              description = "The sheetopia build providing bin/sheetopia-sync.";
            };

            dataDir = lib.mkOption {
              type = lib.types.path;
              default = "/var/lib/sheetopia-sync";
              description = "Directory where the SQLite database and score files are stored.";
            };

            port = lib.mkOption {
              type = lib.types.port;
              default = 8080;
              description = "TCP port the HTTP API listens on.";
            };

            openFirewall = lib.mkOption {
              type = lib.types.bool;
              default = false;
              description = "Whether to open the API port in the host firewall.";
            };
          };

          config =
            lib.mkIf cfg.enable
            {
              nixpkgs.overlays = [ overlay ];
              # Only the admin CLI is installed on the system; the server
              # is reached solely through the unit below.
              environment.systemPackages = [ pkgs.sheetopia-admin ];

              users = {
                users.sheetopia-sync = {
                  isSystemUser = true;
                  group = "sheetopia-sync";
                  description = "sheetopia-sync service user";
                };
                groups.sheetopia-sync = { };
              };

              # Create the data directory with the right owner, works for any
              # dataDir (default or custom).
              systemd.tmpfiles.rules = [
                "d ${cfg.dataDir} 0750 sheetopia-sync sheetopia-sync -"
              ];

              systemd.services.sheetopia-sync = {
                description = "Sheetopia Sync server";
                wantedBy = [ "multi-user.target" ];
                after = [ "network.target" ];
                serviceConfig = {
                  User = "sheetopia-sync";
                  Group = "sheetopia-sync";
                  ExecStart = "${cfg.package}/bin/sheetopia-sync";

                  ReadWritePaths = [ cfg.dataDir ];
                  ProtectSystem = "strict";
                  ProtectHome = true;
                  PrivateTmp = true;

                  Environment = [
                    "DATA_DIR=${cfg.dataDir}"
                    "PORT=${toString cfg.port}"
                  ];
                };
              };

              networking.firewall = lib.mkIf cfg.openFirewall {
                allowedTCPPorts = [ cfg.port ];
              };
            };
        };
    };
}
