{
  lib,
  buildGoModule,
  makeWrapper,
  radicle-node,
  git,
  openssh,
}:
let
  # runtime tools invoked via exec.Command
  runtimeDeps = [
    radicle-node
    git
    openssh
  ];
in
buildGoModule {
  pname = "radicle-mirror";
  version = "0.1.0";
  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      (lib.fileset.fileFilter (file: file.hasExt "go") ../.)
      ../go.mod
      ../gh_pr_event.json
    ];
  };
  vendorHash = null;
  nativeBuildInputs = [ makeWrapper ];
  nativeCheckInputs = [ git ];
  postInstall = ''
    wrapProgram $out/bin/radicle \
      --prefix PATH : ${lib.makeBinPath runtimeDeps}
  '';
  meta = {
    description = "Mirror GitHub repositories to Radicle";
    mainProgram = "radicle";
  };
}
