package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityMiddleware adds security headers and CORS support
func SecurityMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://cdn.arctracker.io;")

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
