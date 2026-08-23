# radicle-mirror

Mirror your GitHub repositories to [Radicle](https://radicle.xyz).

radicle-mirror runs as a GitHub App. When you push to a repository the app is
installed on, it fetches the repository from GitHub and pushes it to a local
Radicle node. It also polls periodically, so missed webhooks or failed syncs
are retried. The Radicle RID for each mirrored repository is reported back to
GitHub as a check run on the pushed commit.

Forks are skipped by default. Forks that should still be mirrored can be
allow-listed with `-mirror-forks` (see below).

## Check runs

Every sync reports a `radicle-mirror` check run on the pushed commit. Its
details link points at a Radicle explorer, configured with `-explorer-url`.
The template replaces `{rid}` with the repository's Radicle ID and `{sha}`
with the pushed commit:

```
-explorer-url 'https://app.radicle.xyz/nodes/seed.radicle.garden/{rid}/commits/{sha}'
```

The linked explorer only shows a commit once its seed node has fetched it.
Use `-rad-connect` to pin seed nodes (`nid@host:port`) the mirror connects to
and prefers for syncing, so the linked instance picks up new commits
immediately. If you run your own explorer against the mirror's storage, links
are live right after a sync without any pinning.

## Setup

1. Create a GitHub App with:
   - Webhook URL pointing at this service's `/github` endpoint (default port
     4128), plus a webhook secret
   - Repository permissions: Contents (read), Checks (write), and optionally
     Variables (read and write) to publish the Radicle RID as a repository
     variable
   - Subscribed to the "Push" event
2. Generate a private key for the App and note the App ID.
3. Create an ed25519 SSH key to use as the Radicle identity:
   `ssh-keygen -t ed25519 -f radicle-key -N ""`
4. Install the App on the repositories you want to mirror.

## Running

```
radicle-mirror \
  -gh-app-id 12345 \
  -gh-app-key-path /run/secrets/gh-app-key.pem \
  -gh-webhook-secret-path /run/secrets/webhook-secret \
  -radicle-key-path /run/secrets/radicle-key \
  -repos-path /var/lib/radicle-mirror/repos \
  -rad-home /var/lib/radicle-mirror/radicle \
  -mirror-forks myorg/some-fork,myorg/another-fork
```

Run `radicle-mirror -help` for all flags (listen address, worker count, sync
timeout, GitHub endpoint).

With Nix:

```
nix run github:Mic92/radicle-mirror -- -help
```

## NixOS module

```nix
{
  imports = [ inputs.radicle-mirror.nixosModules.default ];

  services.radicle-mirror = {
    enable = true;
    ghAppId = 12345;
    ghAppKeyPath = "/run/secrets/gh-app-key.pem";
    webhookSecretPath = "/run/secrets/webhook-secret";
    radicleKeyPath = "/run/secrets/radicle-key";
    # forks are skipped unless listed here
    mirroredForks = [ "myorg/some-fork" ];
    # explorer used for check run details links
    explorerUrl = "https://app.radicle.xyz/nodes/seed.radicle.garden/{rid}/commits/{sha}";
    # seed nodes to connect to and prefer for syncing
    p2pConnect = [ "z6MkrLMMsiPWUcNPHcRajuMi9mDfYckSoJyPwwnknocNYPm7@seed.radicle.garden:8776" ];
  };
}
```

Point the secret paths at files provisioned outside the Nix store, e.g. via
agenix or sops-nix.

## License

[MIT](LICENSE)
