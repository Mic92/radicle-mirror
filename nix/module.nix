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

    mirroredForks = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "Mic92/nixpkgs" ];
      description = "Forks (owner/repo) to mirror. All other forks are skipped.";
    };

    delegates = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "did:key:z6MkjE3BSJn4Y129rhqi5rViSUru8KSBcCQdQcDZq1cnjumw" ];
      description = "Additional DIDs added as delegates on mirrored repositories.";
    };

    allowedOwners = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "Mic92" ];
      description = "GitHub users/orgs whose repositories are mirrored. Empty allows all installations.";
    };

    p2pListen = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "0.0.0.0:8776" ];
      description = "Addresses the radicle node listens on for P2P connections. Empty means outbound-only.";
    };

    p2pExternalAddresses = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "seed.example.com:8776" ];
      description = "External addresses the radicle node announces to peers.";
    };

    explorerUrl = lib.mkOption {
      type = lib.types.str;
      default = "https://app.radicle.xyz/nodes/seed.radicle.garden/{rid}/commits/{sha}";
      description = "URL template for check run details links. {rid} and {sha} are replaced.";
    };

    p2pConnect = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "z6MkrLMMsiPWUcNPHcRajuMi9mDfYckSoJyPwwnknocNYPm7@seed.radicle.garden:8776" ];
      description = "Seed nodes (nid@host:port) the radicle node connects to and prefers for syncing.";
    };

    workers = lib.mkOption {
      type = lib.types.ints.positive;
      default = 4;
      description = "Number of concurrent repository sync workers.";
    };

    syncTimeout = lib.mkOption {
      type = lib.types.str;
      default = "30m";
      description = "Timeout for a single repository sync (Go duration).";
    };

    ridVarName = lib.mkOption {
      type = lib.types.str;
      default = "RADICLE_RID";
      description = ''
        Repository variable the Radicle repository ID is published to. The
        mapping is stored locally, so the actions:write permission is optional.
      '';
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
          "--mirror-forks"
          (lib.concatStringsSep "," cfg.mirroredForks)
          "--delegate"
          (lib.concatStringsSep "," cfg.delegates)
          "--allowed-owners"
          (lib.concatStringsSep "," cfg.allowedOwners)
          "--rad-listen"
          (lib.concatStringsSep "," cfg.p2pListen)
          "--rad-external-address"
          (lib.concatStringsSep "," cfg.p2pExternalAddresses)
          "--rad-connect"
          (lib.concatStringsSep "," cfg.p2pConnect)
          "--explorer-url"
          cfg.explorerUrl
          "--workers"
          (toString cfg.workers)
          "--sync-timeout"
          cfg.syncTimeout
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
