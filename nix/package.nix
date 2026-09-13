{ lib, buildGoModule, makeWrapper, symlinkJoin }:

let
  # unwrapped package that provides both binaries built as is:
  #   $out/bin/sheetopia-sync   (server)
  #   $out/bin/sheetopia-admin  (admin CLI)
  pkg = buildGoModule rec {
    pname = "sheetopia-admin";
    version = "0.2.1-dev";
  
    src = ../.;
  
    subPackages = [ "cmd/server" "cmd/admin" ];
  
    postBuild = ''
      mv $GOPATH/bin/server $GOPATH/bin/sheetopia-sync
      mv $GOPATH/bin/admin $GOPATH/bin/sheetopia-admin
    '';
  
    # Pure-Go sqlite (modernc.org/sqlite): no cgo needed, fully static,
    # matching the upstream Dockerfile (CGO_ENABLED=0).
    env.CGO_ENABLED = 0;
  
    vendorHash = "sha256-PpRnO55eFBmBnSMnJ0ZGgIz7y8DAOzlxT2aedcYRmlk=";
  
    # The database tests (database/user_scoping_test.go) are hermetic:
    # pure-Go sqlite + t.TempDir(), so the default doCheck works in sandbox.
  
    meta = {
      description = "Self-hosted sync server for Sheetopia";
      homepage = "https://github.com/juho05/sheetopia-sync";
      license = lib.licenses.agpl3Only;
      platforms = lib.platforms.linux;
      mainProgram = "sheetopia-sync";
    };
  };
in
# wrapped program to provide the admin CLI with default data directory
symlinkJoin {
    name = "sheetopia-admin";
    inherit (pkg) version;
    paths = [ pkg ];
    nativeBuildInputs = [ makeWrapper ];

    passthru.unwrapped = pkg;

    postBuild = ''
      rm "$out/bin/sheetopia-sync"
      wrapProgram $out/bin/sheetopia-admin \
        --set-default DATA_DIR /var/lib/sheetopia-sync
    '';
  }
