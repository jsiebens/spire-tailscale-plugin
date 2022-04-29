package ts

import (
	"github.com/jsiebens/spire-tailscale-plugin/pkg/common/plugin/ts"
	nodeattestorv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/agent/nodeattestor/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"tailscale.com/client/tailscale"
)

const (
	tokenAudience = ts.TokenAudience
)

type TSAttestorPlugin struct {
	nodeattestorv1.UnimplementedNodeAttestorServer
}

func New() *TSAttestorPlugin {
	return &TSAttestorPlugin{}
}

func (p *TSAttestorPlugin) AidAttestation(stream nodeattestorv1.NodeAttestor_AidAttestationServer) error {
	token, err := tailscale.IDToken(stream.Context(), tokenAudience)
	if err != nil {
		return status.Errorf(codes.Internal, "unable to retrieve valid identity token: %v", err)
	}

	return stream.Send(&nodeattestorv1.PayloadOrChallengeResponse{
		Data: &nodeattestorv1.PayloadOrChallengeResponse_Payload{
			Payload: []byte(token.IDToken),
		},
	})
}
