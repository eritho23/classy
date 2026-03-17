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

  vendorHash = "sha256-58thqQ8XjZwuenniZZuM9N3P0el+w2YLZ9yYItMMUh4=";

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
