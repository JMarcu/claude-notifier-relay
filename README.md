# Claude Notifier Relay

The [Claude Notifier](https://marketplace.visualstudio.com/items?itemName=SingularityInc.claude-notifier) extension for VS Code plays a sound and shows a notification when Claude Code finishes a task, needs permission, or asks a question. As useful as this is, it doesn't work well with dev containers. This repository aims to fix that by providing a relay feature dev containers can include in their build, forwarding the audio cue to the host machine (or another configured target).

## Why the Extension Breaks

By default the extension publishes the audio cue locally. Obviously, this doesn't work inside a container. The extension also includes a `remoteAudio` feature which at first glance might seem like it would solve the problem. However, the feature is more intended for setups such as SSH or WSL.

The `remoteAudio` mode works by establishing a local `cn-daemon` listening on your machine with an SSH reverse port forward to carry the event back from the remote host to the daemon. That bridge assumes a plain SSH session that the daemon can attach a reverse forward to.

Specifically, the issue is that the event's target is hardcoded to `127.0.0.1`, with no host field the daemon or its configuration can redirect. A dev container has its own network namespace nested inside (and possibly behind SSH on top of) the host, so its loopback interface isn't the one the daemon is listening on.

## Solving the Problem

We set up a small TCP proxy to run inside the dev container. It listens on the loopback address the extension is hardcoded to point at (`127.0.0.1`) and forwards each event to the configured host.

Because it listens inside the same network namespace as Claude Code, it *is* the loopback interface the extension is writing to. There's no daemon, SSH session, or port-forwarding rule to set up first. That sidesteps the whole problem described above; the extension's target never has to change, because `cn-relay` is already sitting on it, and its only job is handing the connection off to the host.

## Usage

In your `devcontainer.json` file...

```json
{
    "customizations": {
        "vscode": {
            "extensions": [
                "SingularityInc.claude-notifier"
            ],
            "settings": {
                "claudeNotifier.remoteAudio.enabled": true
            }
        }
    },
    "features": {
        "ghcr.io/jmarcu/claude-notifier-relay/cn-relay:1": {}
    }
}
```

This will:
* Tell the container to include the Claude Notifier extension when it is built.
* Automatically enable `remoteAudio` in the extension's settings, without which it would just output locally within the container.
* Install the relay as a dev container Feature.

If you've changed which port the extension outputs to, you'll need to configure the relay to listen to that same port. You can also configure the relay to output to a different target host. The default values are shown below, but you can set them to whatever you need.

```json
"features": {
    "ghcr.io/jmarcu/claude-notifier-relay/cn-relay:1": {
        "port": "47291",
        "targetHost": "host.docker.internal"
    }
}
```

## Development

### Repo layout

```
cmd/cn-relay/                    Go source for the relay binary
src/cn-relay/
    devcontainer-feature.json    Feature metadata + options
    install.sh                   Runs at image build time
    bin/                         Prebuilt binaries (CI-built, gitignored)
.github/workflows/release.yml    Cross-compiles + publishes on push to main
```

### Releasing

Push to `main` (or run the workflow manually). CI cross-compiles `cmd/cn-relay` for linux/amd64 and linux/arm64, drops the binaries into `src/cn-relay/bin/`, and publishes the feature collection under `src/` to GHCR via [`devcontainers/action`](https://github.com/devcontainers/action). Bump `version` in `devcontainer-feature.json` before tagging a new release — the publish action reads it from there, not from a git tag.
