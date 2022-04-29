package ts

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/hashicorp/hcl"
	"github.com/jsiebens/spire-tailscale-plugin/pkg/common/plugin/ts"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	nodeattestorv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/server/nodeattestor/v1"
	configv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/service/common/config/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sync"
)

const (
	pluginName      = ts.PluginName
	tokenAudience   = ts.TokenAudience
	tailscaleIssuer = "https://login.tailscale.com"
)

type TSAttestorConfig struct {
	trustDomain spiffeid.TrustDomain

	Issuer            string   `hcl:"issuer"`
	DomainIDAllowList []string `hcl:"domain_allow_list"`
}

type TSAttestorPlugin struct {
	nodeattestorv1.UnimplementedNodeAttestorServer
	configv1.UnimplementedConfigServer

	mtx      sync.Mutex
	config   *TSAttestorConfig
	verifier *oidc.IDTokenVerifier
}

func New() *TSAttestorPlugin {
	return &TSAttestorPlugin{}
}

func (p *TSAttestorPlugin) Attest(stream nodeattestorv1.NodeAttestor_AttestServer) error {
	c, err := p.getConfig()
	if err != nil {
		return err
	}

	req, err := stream.Recv()
	if err != nil {
		return err
	}

	payload := req.GetPayload()
	if payload == nil {
		return status.Errorf(codes.InvalidArgument, "missing attestation payload")
	}

	idToken, err := p.verifier.Verify(stream.Context(), string(payload))
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to validate the identity token: %v", err)
	}

	claims := &ts.TailscaleClaims{}
	if err := idToken.Claims(claims); err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to read the identity token claims: %v", err)
	}

	domainMatchesAllowList := false
	for _, domain := range c.DomainIDAllowList {
		if claims.Domain == domain {
			domainMatchesAllowList = true
			break
		}
	}
	if !domainMatchesAllowList {
		return status.Errorf(codes.PermissionDenied, "identity token project ID %q is not in the allow list", claims.Domain)
	}

	id, err := agentID(c.trustDomain, fmt.Sprintf("/%s/%s", pluginName, claims.Node))
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

func (p *TSAttestorPlugin) Configure(ctx context.Context, req *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error) {
	hclConfig := new(TSAttestorConfig)
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

	if len(hclConfig.DomainIDAllowList) == 0 {
		return nil, status.Error(codes.InvalidArgument, "domain_allow_list is required")
	}

	if hclConfig.Issuer == "" {
		hclConfig.Issuer = tailscaleIssuer
	}

	provider, err := oidc.NewProvider(ctx, hclConfig.Issuer)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "issuer is invalid: %v", err)
	}

	hclConfig.trustDomain = trustDomain

	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.config = hclConfig
	p.verifier = provider.Verifier(&oidc.Config{ClientID: tokenAudience})

	return &configv1.ConfigureResponse{}, nil
}

func (p *TSAttestorPlugin) getConfig() (*TSAttestorConfig, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

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
