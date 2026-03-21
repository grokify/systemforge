package bff

import (
	"net"
	"net/http"
	"strings"
)

// ClientIPConfig contains configuration for client IP extraction.
type ClientIPConfig struct {
	// TrustCloudflare enables Cloudflare header support.
	// When true, checks CF-Connecting-IP and True-Client-IP headers.
	// Default: false.
	TrustCloudflare bool

	// TrustProxy enables standard proxy headers (X-Forwarded-For, X-Real-IP).
	// Only enable if behind a trusted reverse proxy.
	// Default: false.
	TrustProxy bool

	// TrustedProxies is a list of trusted proxy IP ranges.
	// If set, proxy headers are only trusted when the request comes from these IPs.
	// Supports CIDR notation (e.g., "10.0.0.0/8", "172.16.0.0/12").
	TrustedProxies []string

	// CloudflareIPRanges can be set to validate that CF headers come from Cloudflare.
	// If empty and TrustCloudflare is true, CF headers are trusted unconditionally.
	// Cloudflare publishes their IP ranges at:
	// https://www.cloudflare.com/ips-v4 and https://www.cloudflare.com/ips-v6
	CloudflareIPRanges []string

	trustedNets    []*net.IPNet
	cloudflareNets []*net.IPNet
}

// DefaultClientIPConfig returns a safe default configuration.
// By default, no proxy headers are trusted.
func DefaultClientIPConfig() ClientIPConfig {
	return ClientIPConfig{
		TrustCloudflare: false,
		TrustProxy:      false,
	}
}

// CloudflareClientIPConfig returns configuration for Cloudflare deployments.
func CloudflareClientIPConfig() ClientIPConfig {
	return ClientIPConfig{
		TrustCloudflare: true,
		TrustProxy:      true, // Cloudflare also sets X-Forwarded-For
	}
}

// ClientIPExtractor extracts the real client IP from requests.
type ClientIPExtractor struct {
	config ClientIPConfig
}

// NewClientIPExtractor creates a new client IP extractor.
func NewClientIPExtractor(config ClientIPConfig) *ClientIPExtractor {
	e := &ClientIPExtractor{config: config}

	// Parse trusted proxy ranges
	for _, cidr := range config.TrustedProxies {
		if _, ipNet, err := net.ParseCIDR(cidr); err == nil {
			e.config.trustedNets = append(e.config.trustedNets, ipNet)
		} else if ip := net.ParseIP(cidr); ip != nil {
			// Single IP, convert to /32 or /128
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			e.config.trustedNets = append(e.config.trustedNets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
		}
	}

	// Parse Cloudflare IP ranges
	for _, cidr := range config.CloudflareIPRanges {
		if _, ipNet, err := net.ParseCIDR(cidr); err == nil {
			e.config.cloudflareNets = append(e.config.cloudflareNets, ipNet)
		}
	}

	return e
}

// GetClientIP extracts the real client IP from the request.
// It checks headers in order of trust: Cloudflare > Proxy > RemoteAddr.
func (e *ClientIPExtractor) GetClientIP(r *http.Request) string {
	remoteIP := extractIP(r.RemoteAddr)

	// Check Cloudflare headers if enabled
	if e.config.TrustCloudflare && e.isCloudflareRequest(remoteIP) {
		// CF-Connecting-IP is the most reliable
		if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
			return ip
		}
		// True-Client-IP is available on Enterprise plans
		if ip := r.Header.Get("True-Client-IP"); ip != "" {
			return ip
		}
	}

	// Check standard proxy headers if enabled
	if e.config.TrustProxy && e.isTrustedProxy(remoteIP) {
		// X-Real-IP is typically set by nginx
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return ip
		}
		// X-Forwarded-For contains a chain; the first untrusted IP is the client
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return e.parseXForwardedFor(xff)
		}
	}

	return remoteIP
}

// isCloudflareRequest checks if the request comes from Cloudflare.
func (e *ClientIPExtractor) isCloudflareRequest(remoteIP string) bool {
	// If no Cloudflare ranges configured, trust unconditionally (user's choice)
	if len(e.config.cloudflareNets) == 0 {
		return true
	}

	ip := net.ParseIP(remoteIP)
	if ip == nil {
		return false
	}

	for _, ipNet := range e.config.cloudflareNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// isTrustedProxy checks if the remote IP is a trusted proxy.
func (e *ClientIPExtractor) isTrustedProxy(remoteIP string) bool {
	// If no trusted proxies configured, trust unconditionally (user's choice)
	if len(e.config.trustedNets) == 0 {
		return true
	}

	ip := net.ParseIP(remoteIP)
	if ip == nil {
		return false
	}

	for _, ipNet := range e.config.trustedNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// parseXForwardedFor extracts the client IP from X-Forwarded-For header.
// Returns the leftmost (first) IP, which is typically the original client.
func (e *ClientIPExtractor) parseXForwardedFor(xff string) string {
	// X-Forwarded-For: client, proxy1, proxy2
	ips := strings.Split(xff, ",")
	if len(ips) == 0 {
		return ""
	}

	// If we have trusted proxy ranges, find the first untrusted IP from the right
	if len(e.config.trustedNets) > 0 {
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			if !e.isTrustedProxy(ip) {
				return ip
			}
		}
	}

	// Otherwise return the leftmost IP
	return strings.TrimSpace(ips[0])
}

// extractIP extracts the IP address from a host:port string.
func extractIP(addr string) string {
	// Handle IPv6 addresses like [::1]:8080
	if strings.HasPrefix(addr, "[") {
		if idx := strings.Index(addr, "]:"); idx != -1 {
			return addr[1:idx]
		}
		return strings.Trim(addr, "[]")
	}

	// Handle IPv4 addresses like 127.0.0.1:8080
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}

	return addr
}

// GetCloudflareMetadata extracts Cloudflare-specific metadata from the request.
func GetCloudflareMetadata(r *http.Request) map[string]string {
	metadata := make(map[string]string)

	// Ray ID for request tracing
	if ray := r.Header.Get("CF-Ray"); ray != "" {
		metadata["cf_ray"] = ray
	}

	// Country code
	if country := r.Header.Get("CF-IPCountry"); country != "" {
		metadata["cf_country"] = country
	}

	// Visitor scheme (http/https)
	if scheme := r.Header.Get("CF-Visitor"); scheme != "" {
		metadata["cf_visitor"] = scheme
	}

	// Bot detection (Enterprise)
	if botScore := r.Header.Get("CF-Bot-Score"); botScore != "" {
		metadata["cf_bot_score"] = botScore
	}

	return metadata
}
