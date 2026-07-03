{
  mkShell,
  go,
  gotools,
  radicle-node,
  git,
  openssh,
}:
mkShell {
  packages = [
    go
    gotools
    radicle-node
    git
    openssh
  ];
}
