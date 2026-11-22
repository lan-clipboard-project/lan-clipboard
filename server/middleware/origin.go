package middleware

import (
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"lan-clipboard/config"

	"github.com/gin-gonic/gin"
)

// AllowedHost limits the authority seen by browsers and prevents unconfigured
// hostnames from reaching the server, including after a DNS rebinding.
func AllowedHost(publicOrigin *url.URL) gin.HandlerFunc {
	return func(c *gin.Context) {
		host := (&url.URL{Host: c.Request.Host}).Hostname()
		ip, err := netip.ParseAddr(host)
		ip = ip.Unmap()
		local := strings.EqualFold(host, "localhost") ||
			(err == nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()))
		if !local && !matchesPublicHost(c.Request.Host, publicOrigin) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "host not allowed"})
		}
	}
}

func SameOrigin(publicOrigin *url.URL) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !BrowserOriginAllowed(c.Request, publicOrigin) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "browser origin not allowed"})
		}
	}
}

// BrowserOriginAllowed applies to the API and WebSocket upgrade. CLI clients and
// container health checks can call the service without browser headers.
func BrowserOriginAllowed(r *http.Request, publicOrigin *url.URL) bool {
	for _, site := range r.Header.Values("Sec-Fetch-Site") {
		if site == "cross-site" || site == "same-site" {
			return false
		}
	}
	origins := r.Header.Values("Origin")
	if len(origins) == 0 {
		return true
	}
	if len(origins) != 1 {
		return false
	}
	origin, err := config.ParseOrigin(origins[0])
	if err != nil {
		return false
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if matchesPublicHost(r.Host, publicOrigin) {
		scheme = publicOrigin.Scheme
	}
	return origin.Scheme == scheme && sameAuthority(r.Host, origin.Host, scheme)
}

func matchesPublicHost(host string, publicOrigin *url.URL) bool {
	return publicOrigin != nil && sameAuthority(host, publicOrigin.Host, publicOrigin.Scheme)
}

func sameAuthority(left, right, scheme string) bool {
	a, b := &url.URL{Host: left}, &url.URL{Host: right}
	return strings.EqualFold(a.Hostname(), b.Hostname()) && originPort(a, scheme) == originPort(b, scheme)
}

func originPort(authority *url.URL, scheme string) string {
	if port := authority.Port(); port != "" {
		return port
	}
	if scheme == "https" {
		return "443"
	}
	return "80"
}
