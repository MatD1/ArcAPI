package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityMiddleware adds security headers and CORS support
func SecurityMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Supabase URL from environment for CSP
		supabaseURL := os.Getenv("NEXT_PUBLIC_SUPABASE_URL")
		// Get Appwrite endpoint from environment for CSP (check both env var names)
		appwriteEndpoint := os.Getenv("APPWRITE_ENDPOINT")
		if appwriteEndpoint == "" {
			appwriteEndpoint = os.Getenv("NEXT_PUBLIC_APPWRITE_ENDPOINT")
		}
		
		// Build CSP policy
		// Note: frame-ancestors allows embedding (for OAuth redirects), form-action allows form submissions
		// frame-src allows iframes (for OAuth flows), connect-src allows fetch/XHR requests
		connectSrc := "'self'"
		frameSrc := "'self'"
		
		// Add Supabase to CSP
		if supabaseURL != "" {
			// Parse URL to extract scheme and host
			parsedURL, err := url.Parse(supabaseURL)
			if err == nil {
				// Allow the full Supabase URL domain
				domain := parsedURL.Scheme + "://" + parsedURL.Host
				connectSrc += " " + domain
				// Also allow common Supabase patterns
				connectSrc += " https://*.supabase.co https://*.supabase.in"
			} else {
				// If parsing fails, just add the URL as-is
				connectSrc += " " + supabaseURL + " https://*.supabase.co https://*.supabase.in"
			}
		}
		
		// Always allow Appwrite cloud instances (for OAuth redirects and API calls)
		// This ensures it works even if endpoint is loaded dynamically from API config
		// CSP wildcards only match one level, so we need multiple patterns for different Appwrite regions
		// Allow all Appwrite cloud instances (e.g., syd.cloud.appwrite.io, fra.cloud.appwrite.io, etc.)
		// Pattern: *.cloud.appwrite.io matches any subdomain of cloud.appwrite.io (e.g., syd.cloud.appwrite.io)
		connectSrc += " https://*.appwrite.io https://*.cloud.appwrite.io https://appwrite.io"
		frameSrc += " https://*.appwrite.io https://*.cloud.appwrite.io https://appwrite.io"
		
		// Also allow common Appwrite cloud regions explicitly (as a fallback)
		// This ensures coverage even if wildcard matching has issues
		commonRegions := []string{
			"syd.cloud.appwrite.io", "fra.cloud.appwrite.io", "nyc.cloud.appwrite.io",
			"lon.cloud.appwrite.io", "tor.cloud.appwrite.io", "sgp.cloud.appwrite.io",
			"ams.cloud.appwrite.io", "blr.cloud.appwrite.io", "waw.cloud.appwrite.io",
			"iad.cloud.appwrite.io", "cle.cloud.appwrite.io", "dub.cloud.appwrite.io",
		}
		for _, region := range commonRegions {
			connectSrc += " https://" + region
			frameSrc += " https://" + region
		}
		
		// Add specific Appwrite endpoint if configured (for self-hosted instances)
		if appwriteEndpoint != "" {
			// Parse URL to extract scheme and host
			parsedURL, err := url.Parse(appwriteEndpoint)
			if err == nil {
				// Allow the full Appwrite endpoint domain
				domain := parsedURL.Scheme + "://" + parsedURL.Host
				connectSrc += " " + domain
				frameSrc += " " + domain
			} else {
				// If parsing fails, just add the URL as-is
				connectSrc += " " + appwriteEndpoint
				frameSrc += " " + appwriteEndpoint
			}
		}
		
		csp := fmt.Sprintf("default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://cdn.arctracker.io; connect-src %s; frame-src %s; frame-ancestors 'self'; form-action 'self'", connectSrc, frameSrc)
		
		csp += ";"

		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", csp)

		// CORS headers
		origin := c.GetHeader("Origin")
		if origin != "" {
			allowed := false

			// Check if origin is in allowed list
			if len(allowedOrigins) > 0 {
				for _, allowedOrigin := range allowedOrigins {
					if origin == allowedOrigin {
						allowed = true
						break
					}
				}
			} else {
				// If no origins configured, allow localhost and same origin for development
				if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
					allowed = true
				}
			}

			if allowed {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Requested-With")
				c.Header("Access-Control-Max-Age", "3600")
			}

			// Handle preflight requests
			if c.Request.Method == http.MethodOptions {
				if allowed {
					c.AbortWithStatus(http.StatusNoContent)
				} else {
					c.AbortWithStatus(http.StatusForbidden)
				}
				return
			}
		}

		// HTTPS enforcement (only in production)
		if c.Request.TLS == nil && strings.HasPrefix(c.Request.Proto, "HTTP/") {
			// Check X-Forwarded-Proto header (for proxies like Railway)
			if c.GetHeader("X-Forwarded-Proto") != "https" && c.GetHeader("X-Forwarded-Proto") != "" {
				// Only redirect if not in development
				if !strings.Contains(c.Request.Host, "localhost") && !strings.Contains(c.Request.Host, "127.0.0.1") {
					c.Redirect(http.StatusPermanentRedirect, "https://"+c.Request.Host+c.Request.RequestURI)
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}
