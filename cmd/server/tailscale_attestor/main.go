package main

import (
	"context"
	"fmt"
	"sync"
	"tailscale.com/util/dnsname"

	"github.com/hashicorp/hcl"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/spire-plugin-sdk/pluginmain"
	nodeattestorv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/server/nodeattestor/v1"
	configv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/service/common/config/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/types/key"
)

const (
	PluginName = "ts"
)

type Config struct {
	trustDomain spiffeid.TrustDomain
}

// Plugin implements the NodeAttestor plugin
type Plugin struct {
	nodeattestorv1.UnimplementedNodeAttestorServer
	configv1.UnimplementedConfigServer

	configMtx sync.RWMutex
	config    *Config
}

func (p *Plugin) Attest(stream nodeattestorv1.NodeAttestor_AttestServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	c, err := p.getConfig()
	if err != nil {
		return err
	}

	payload := req.GetPayload()

	clientKey := key.NodePublic{}
	if err := clientKey.UnmarshalText(payload); err != nil {
		return err
	}

	tsStatus, err := tailscale.Status(stream.Context())
	if err != nil {
		return fmt.Errorf("failed to query local tailscaled status: %w", err)
	}

	var node *ipnstate.PeerStatus

	if clientKey == tsStatus.Self.PublicKey {
		node = tsStatus.Self
	}

	if p, exists := tsStatus.Peer[clientKey]; exists {
		node = p
	}

	if node == nil {
		return fmt.Errorf("unable to find provided client key")
	}

	sanitizeHostname := dnsname.SanitizeHostname(node.HostName)
	id, err := agentID(c.trustDomain, fmt.Sprintf("/%s/%s", PluginName, sanitizeHostname))
	if err != nil {
		return err
	}

	return stream.Send(&nodeattestorv1.AttestResponse{
		Response: &nodeattestorv1.AttestResponse_AgentAttributes{
			AgentAttributes: &nodeattestorv1.AgentAttributes{
				SpiffeId: id.String(),
			},
		},
	})
}

func (p *Plugin) Configure(ctx context.Context, req *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error) {
	hclConfig := new(Config)
	if err := hcl.Decode(hclConfig, req.HclConfiguration); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unable to decode configuration: %v", err)
	}

	if req.CoreConfiguration == nil {
		return nil, status.Error(codes.InvalidArgument, "global configuration is required")
	}

	if req.CoreConfiguration.TrustDomain == "" {
		return nil, status.Error(codes.InvalidArgument, "trust_domain is required")
	}

	trustDomain, err := spiffeid.TrustDomainFromString(req.CoreConfiguration.TrustDomain)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "trust_domain is invalid: %v", err)
	}

	hclConfig.trustDomain = trustDomain

	p.setConfig(hclConfig)
	return &configv1.ConfigureResponse{}, nil
}

func (p *Plugin) setConfig(config *Config) {
	p.configMtx.Lock()
	p.config = config
	p.configMtx.Unlock()
}

func (p *Plugin) getConfig() (*Config, error) {
	p.configMtx.RLock()
	defer p.configMtx.RUnlock()
	if p.config == nil {
		return nil, status.Error(codes.FailedPrecondition, "not configured")
	}
	return p.config, nil
}

func agentID(td spiffeid.TrustDomain, suffix string) (spiffeid.ID, error) {
	if td.IsZero() {
		return spiffeid.ID{}, fmt.Errorf("cannot create agent ID with suffix %q for empty trust domain", suffix)
	}
	if err := spiffeid.ValidatePath(suffix); err != nil {
		return spiffeid.ID{}, fmt.Errorf("invalid agent path suffix %q: %w", suffix, err)
	}
	return spiffeid.FromPath(td, "/spire/agent"+suffix)
}

func main() {
	plugin := new(Plugin)
	pluginmain.Serve(
		nodeattestorv1.NodeAttestorPluginServer(plugin),
		configv1.ConfigServiceServer(plugin),
	)
}
