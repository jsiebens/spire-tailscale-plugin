package main

import (
	"github.com/spiffe/spire-plugin-sdk/pluginmain"
	nodeattestorv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/agent/nodeattestor/v1"
	"tailscale.com/client/tailscale"
)

type Plugin struct {
	nodeattestorv1.UnimplementedNodeAttestorServer
}

func (p *Plugin) AidAttestation(stream nodeattestorv1.NodeAttestor_AidAttestationServer) error {
	status, err := tailscale.Status(stream.Context())
	if err != nil {
		return err
	}

	payload, err := status.Self.PublicKey.MarshalText()
	if err != nil {
		return err
	}

	return stream.Send(&nodeattestorv1.PayloadOrChallengeResponse{
		Data: &nodeattestorv1.PayloadOrChallengeResponse_Payload{
			Payload: payload,
		},
	})
}

func main() {
	plugin := new(Plugin)
	pluginmain.Serve(
		nodeattestorv1.NodeAttestorPluginServer(plugin),
	)
}
