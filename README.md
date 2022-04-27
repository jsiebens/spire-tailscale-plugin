# SPIRE Tailscale Plugin

> :warning:  this project is still WIP and experimental (see [#1](https://github.com/jsiebens/spire-tailscale-plugin/issues/1))
 
This repository contains agent and server plugins for [SPIRE](https://github.com/spiffe/spire) to allow [Tailscale](https://tailscale.com) node attestation.

## Quick Start

Before starting, create a running SPIRE deployment and add the following configuration to the agent and server.
Both server and agents should be running on a Tailscale node.

### Agent Configuration

```hcl
NodeAttestor "tailscale" {
	plugin_cmd = "/path/to/plugin_cmd"
	plugin_checksum = "sha256 of the plugin binary"
	plugin_data {
	}
}
```

### Server Configuration

```hcl
NodeAttestor "tailscale" {
	plugin_cmd = "/path/to/plugin_cmd"
	plugin_checksum = "sha256 of the plugin binary"
	plugin_data {
	}
}
```

## How it Works

The plugin uses the Tailscale Node public keys as the method of attestation and is inspired on the [client verification](https://tailscale.com/kb/1118/custom-derp-servers/?q=derp#optional-restricting-client-access-to-your-derp-node) in custom DERP servers.
The plugin operates as follows:

1. Agent fetches the Tailscale Node key from the local `tailscaled` agent
1. Agent sends the key to the server
1. Server inspects the key and checks if it is a valid key in its Tailscale network.
1. Server creates a SPIFFE ID in the form of `spiffe://<trust_domain>/spire/agent/ts/<hostname>`
1. All done!
