{
  buildGoModule,
  rev ? "unknown-revision",
  lib,
  sqlc,
  templ,
  pdpmake,
  minify,
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
    minify
  ];

  preBuild = ''
    pdpmake templ sqlc
  '';

  postInstall = ''
    mkdir -p "$out/static"
    minify ./static/stylesheet.css > "$out/static/stylesheet.css"
    cp static/*.txt "$out/static"
  '';

  meta = {
    mainProgram = "classy";
  };
}
