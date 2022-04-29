# SPIRE Tailscale Plugin

> :warning: this node attestation plugin relies on a Tailscale OIDC id-token feature, which is marked as Work-in-Progress and may not be available for everyone yet. 
 
This repository contains agent and server plugins for [SPIRE](https://github.com/spiffe/spire) to allow [Tailscale](https://tailscale.com) node attestation.

## Quick Start

Before starting, create a running SPIRE deployment and add the following configuration to the agent and server.
The agents should be running on a Tailscale node, with version __>= 1.24.0__.

### Agent Configuration

```hcl
NodeAttestor "tailscale" {
  plugin_cmd = "/path/to/plugin_cmd"
  plugin_checksum = "sha256 of the plugin binary"
  plugin_data {
    domain_allow_list = [ "example.com" ]
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

This plugin automatically attests instances using the Tailscale OIDC Token (a Tailscale feature still in WIP), and operates as follows:

1. Agent fetches a Tailscale OIDC token from the local `tailscaled` agent
1. Agent sends the token to the server
1. Server validates the token.
1. Server creates a SPIFFE ID in the form of `spiffe://<trust_domain>/spire/agent/tailscale/<hostname>`
1. All done!
