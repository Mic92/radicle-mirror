# radicle-mirror

Mirror your GitHub repositories to [Radicle](https://radicle.xyz).

radicle-mirror runs as a GitHub App. When you push to a repository the app is
installed on, it fetches the repository from GitHub and pushes it to a local
Radicle node. It also polls periodically, so missed webhooks or failed syncs
are retried. The Radicle RID for each mirrored repository is reported back to
GitHub as a check run on the pushed commit.

## Setup

1. Create a GitHub App with:
   - Webhook URL pointing at this service (default port 4128), plus a webhook secret
   - Repository permissions: Contents (read), Checks (write)
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
  -rad-home /var/lib/radicle-mirror/radicle
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
  };
}
```

Point the secret paths at files provisioned outside the Nix store, e.g. via
agenix or sops-nix.

## License

[MIT](LICENSE)
