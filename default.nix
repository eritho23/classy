{
  buildGoModule,
  rev ? "unknown-revision",
  lib,
  sqlc,
  templ,
  pdpmake,
  minify,
  openssl,
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
    minify
    openssl
    pdpmake
    sqlc
    templ
  ];

  preBuild = ''
    pdpmake templ sqlc
  '';

  postInstall = ''
    mkdir -p "$out/static"
    minify ./static/stylesheet.css > "$out/static/stylesheet.css"
    cp static/*.txt "$out/static"

    touch "$out/caddy_csp_config_snippet"
    STYLESHEET_HASH="$(cat "$out/static/stylesheet.css" | openssl dgst -sha256 -binary | base64)"

    cat > "$out/caddy_csp_config_snippet" <<EOF
    header Content-Security-Policy "default-src 'self'; style-src 'self' 'sha256-$STYLESHEET_HASH'"
    EOF
  '';

  meta = {
    mainProgram = "classy";
  };
}
