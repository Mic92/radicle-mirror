{
  pkgs,
  self,
}:
let
  appId = 12345;
  installationId = 1;
  ownerLogin = "testorg";
  ownerId = 100;
  repoId = 200;
  repoName = "testrepo";
  repoFullName = "${ownerLogin}/${repoName}";
  webhookSecret = "topsecret";

  # GitHub App keys are distributed as PKCS#1 PEM
  ghAppKey = pkgs.runCommand "gh-app-key.pem" { nativeBuildInputs = [ pkgs.openssl ]; } ''
    openssl genrsa -traditional -out "$out" 2048
  '';

  radicleKey = pkgs.runCommand "radicle-key" { nativeBuildInputs = [ pkgs.openssh ]; } ''
    ssh-keygen -t ed25519 -N "" -C radicle -f "$out"
    rm "$out.pub"
  '';

  webhookSecretFile = pkgs.writeText "webhook-secret" webhookSecret;

  cloneHost = "githost";
  clonePort = 8443;
  cloneUrl = "https://${cloneHost}:${toString clonePort}/${repoName}.git";

  tlsCert = pkgs.runCommand "githost-cert" { nativeBuildInputs = [ pkgs.openssl ]; } ''
    mkdir -p "$out"
    openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
      -keyout "$out/key.pem" -out "$out/cert.pem" \
      -subj "/CN=${cloneHost}" -addext "subjectAltName=DNS:${cloneHost}"
  '';

  # a bare repo laid out for dumb-http clone (update-server-info)
  webRoot = pkgs.runCommand "gitroot" { nativeBuildInputs = [ pkgs.git ]; } ''
    export HOME=$TMPDIR
    git config --global user.email test@example.com
    git config --global user.name test
    git config --global init.defaultBranch main
    git init -q work
    ( cd work && echo hello > file.txt && git add file.txt && git commit -qm "initial commit" )
    mkdir -p "$out"
    git clone -q --bare work "$out/${repoName}.git"
    git -C "$out/${repoName}.git" update-server-info
  '';
in
pkgs.testers.runNixOSTest {
  name = "radicle-mirror";

  nodes.machine =
    { ... }:
    {
      imports = [ self.nixosModules.default ];

      networking.hosts."127.0.0.1" = [ cloneHost ];
      security.pki.certificateFiles = [ "${tlsCert}/cert.pem" ];

      services.radicle-mirror = {
        enable = true;
        package = self.packages.${pkgs.system}.default;
        ghAppId = appId;
        ghAppKeyPath = "${ghAppKey}";
        webhookSecretPath = "${webhookSecretFile}";
        radicleKeyPath = "${radicleKey}";
        ghEndpoint = "http://127.0.0.1:3000/";
        inherit cloneHost;
      };

      systemd.services.git-https = {
        description = "HTTPS git server";
        wantedBy = [ "multi-user.target" ];
        before = [ "radicle-mirror.service" ];
        serviceConfig = {
          ExecStart = "${pkgs.python3}/bin/python3 ${./tls-fileserver.py} ${webRoot} ${tlsCert}/cert.pem ${tlsCert}/key.pem ${toString clonePort}";
          DynamicUser = true;
        };
      };

      systemd.services.fake-github = {
        description = "Fake GitHub API";
        wantedBy = [ "multi-user.target" ];
        before = [ "radicle-mirror.service" ];
        environment = {
          APP_ID = toString appId;
          INSTALLATION_ID = toString installationId;
          OWNER_LOGIN = ownerLogin;
          OWNER_ID = toString ownerId;
          REPO_ID = toString repoId;
          REPO_NAME = repoName;
          REPO_FULLNAME = repoFullName;
          CLONE_URL = cloneUrl;
          RID_FILE = "/var/lib/fake-github/rid";
          CHECK_RUN_FILE = "/var/lib/fake-github/check-run";
        };
        serviceConfig = {
          ExecStart = "${pkgs.python3}/bin/python3 ${./fake-github.py}";
          StateDirectory = "fake-github";
          DynamicUser = true;
        };
      };
    };

  testScript = ''
    import base64, hashlib, hmac, json

    secret = ${builtins.toJSON webhookSecret}
    head_sha = "a" * 40
    payload = json.dumps({
        "ref": "refs/heads/main",
        "after": head_sha,
        "repository": {
            "id": ${toString repoId},
            "name": ${builtins.toJSON repoName},
            "full_name": ${builtins.toJSON repoFullName},
            "pushed_at": 1782023169,
            "description": "integration test repo",
            "private": False,
            "owner": {"login": ${builtins.toJSON ownerLogin}, "id": ${toString ownerId}},
            "clone_url": ${builtins.toJSON cloneUrl},
        },
    })
    sig = "sha256=" + hmac.new(secret.encode(), payload.encode(), hashlib.sha256).hexdigest()

    machine.start()
    machine.wait_for_unit("fake-github.service")
    machine.wait_for_unit("radicle-mirror.service")
    machine.wait_for_open_port(3000)
    machine.wait_for_open_port(4128)
    machine.wait_for_open_port(${toString clonePort})

    machine.succeed("curl -fsS http://127.0.0.1:4128/health")

    b64 = base64.b64encode(payload.encode()).decode()
    machine.succeed(f"echo {b64} | base64 -d > /tmp/payload")
    machine.succeed(
        "curl -fsS -X POST http://127.0.0.1:4128/github "
        + f"-H 'X-Hub-Signature-256: {sig}' "
        + "-H 'X-GitHub-Event: push' "
        + "--data-binary @/tmp/payload"
    )

    # the RID is written back via the repo variable PATCH only after a successful
    # clone + rad init + push to Radicle
    machine.wait_until_succeeds("test -s /var/lib/fake-github/rid", timeout=60)
    rid = machine.succeed("cat /var/lib/fake-github/rid").strip()
    assert rid.startswith("rad:"), f"unexpected rid: {rid!r}"

    # a successful mirror reports a completed check run for the pushed commit
    machine.wait_until_succeeds("test -s /var/lib/fake-github/check-run", timeout=60)
    check = json.loads(machine.succeed("cat /var/lib/fake-github/check-run"))
    assert check["head_sha"] == head_sha, check
    assert check["status"] == "completed", check
    assert check["conclusion"] == "success", check
  '';
}
