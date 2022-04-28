package main

import (
	"github.com/jsiebens/spire-tailscale-plugin/pkg/agent/plugin/nodeattestor/ts"
	"github.com/spiffe/spire-plugin-sdk/pluginmain"
	nodeattestorv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/agent/nodeattestor/v1"
)

func main() {
	plugin := ts.New()
	pluginmain.Serve(
		nodeattestorv1.NodeAttestorPluginServer(plugin),
	)
}
