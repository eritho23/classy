{
  buildGoModule,
  rev ? "unknown-revision",
  lib,
  sqlc,
  templ,
  pdpmake,
}:
buildGoModule {
  pname = "classy";
  version = rev;

  src = lib.cleanSource ./.;

  vendorHash = "sha256-D4EGzxCZLuh0WRIN/QjwHM5/FAr9GwYv1+SJzMvLJpo=";

  subPackages = [
    "cmd/classy"
  ];

  nativeBuildInputs = [
    sqlc
    templ
    pdpmake
  ];

  preBuild = ''
    sqlc generate
    sqlc generate
    pdpmake templ sqlc
  '';

  meta = {
    mainProgram = "classy";
  };
}
