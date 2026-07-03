{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.services.radicle-mirror;
in
{
  options.services.radicle-mirror = {
    enable = lib.mkEnableOption "the GitHub to Radicle mirror service";

    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.callPackage ./package.nix { };
      defaultText = lib.literalExpression "pkgs.radicle-mirror";
      description = "The radicle-mirror package to use.";
    };

    addr = lib.mkOption {
      type = lib.types.str;
      default = ":4128";
      description = "Address the HTTP server listens on.";
    };

    ghAppId = lib.mkOption {
      type = lib.types.int;
      description = "GitHub App ID.";
    };

    # secrets use str, not path, so a runtime path is not copied into the
    # world-readable Nix store
    ghAppKeyPath = lib.mkOption {
      type = lib.types.str;
      example = "/run/secrets/gh-app-key.pem";
      description = "Runtime path to the GitHub App RSA private key file (PKCS#1 or PKCS#8, PEM or DER).";
    };

    webhookSecretPath = lib.mkOption {
      type = lib.types.str;
      example = "/run/secrets/webhook-secret";
      description = "Runtime path to the GitHub webhook secret file.";
    };

    radicleKeyPath = lib.mkOption {
      type = lib.types.str;
      example = "/run/secrets/radicle-key";
      description = "Runtime path to the Radicle (OpenSSH ed25519) private key file.";
    };

    ghEndpoint = lib.mkOption {
      type = lib.types.str;
      default = "https://api.github.com/";
      description = "GitHub API endpoint to contact.";
    };

    cloneHost = lib.mkOption {
      type = lib.types.str;
      default = "github.com";
      description = "Host that repositories may be cloned from over https.";
    };

    ridVarName = lib.mkOption {
      type = lib.types.str;
      default = "RADICLE_RID";
      description = "Repository variable used to store the Radicle repository ID.";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.radicle-mirror = {
      description = "GitHub to Radicle mirror";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];

      serviceConfig = {
        ExecStart = lib.escapeShellArgs [
          (lib.getExe cfg.package)
          "--addr"
          cfg.addr
          "--gh-app-id"
          (toString cfg.ghAppId)
          "--gh-app-key-path"
          cfg.ghAppKeyPath
          "--gh-webhook-secret-path"
          cfg.webhookSecretPath
          "--gh-endpoint"
          cfg.ghEndpoint
          "--gh-clone-host"
          cfg.cloneHost
          "--gh-rid-var-name"
          cfg.ridVarName
          "--radicle-key-path"
          cfg.radicleKeyPath
          "--repos-path"
          "%S/radicle-mirror/repos"
          "--rad-home"
          "%S/radicle-mirror/rad"
        ];
        Restart = "on-failure";
        RestartSec = 5;

        DynamicUser = true;
        StateDirectory = "radicle-mirror";
        WorkingDirectory = "%S/radicle-mirror";

        # hardening
        NoNewPrivileges = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        PrivateTmp = true;
        PrivateDevices = true;
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
        ];
        RestrictNamespaces = true;
        LockPersonality = true;
        MemoryDenyWriteExecute = true;
        SystemCallArchitectures = "native";
      };
    };
  };
}
