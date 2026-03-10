{
  buildGoModule,
  rev ? "unknown-revision",
  lib,
  sqlc,
}:
buildGoModule {
  pname = "gopodder";
  version = rev;

  src = lib.cleanSource ./.;

  vendorHash = "sha256-vSKqT/ICgWH0KemTyGbuimEUNpFck+RXwkmSC73lpI8=";

  subPackages = [
    "cmd/classy"
  ];

  nativeBuildInputs = [
    sqlc
  ];

  preBuild = ''
    sqlc generate
  '';

  meta = {
    mainProgram = "classy";
  };
}
