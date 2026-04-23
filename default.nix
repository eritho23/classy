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

  vendorHash = "sha256-rumU/gg0Ln7j8CicF1c7wyT0Qv8VHOWlr5KexW3nlNM=";

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
