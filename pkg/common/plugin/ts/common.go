package ts

const (
	PluginName    = "tailscale"
	TokenAudience = "spire-tailscale-node-attestor"
)

type TailscaleClaims struct {
	Node   string `json:"node"`
	Domain string `json:"domain"`
}
