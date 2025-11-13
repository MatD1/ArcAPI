package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// LoggerMiddleware logs all requests to the audit_logs table
func LoggerMiddleware(auditLogRepo *repository.AuditLogRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Read request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Capture response
		w := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = w

		// Process request
		c.Next()

		// Calculate response time
		responseTime := time.Since(start).Milliseconds()

		// Get auth context
		var apiKeyID, jwtTokenID, userID *uint
		if authCtx, exists := c.Get(AuthContextKey); exists {
			ctx := authCtx.(*AuthContext)
			if key, ok := ctx.APIKey.(*models.APIKey); ok {
				apiKeyID = &key.ID
				if key.UserID != 0 {
					userID = &key.UserID
				}
			}
			// JWT token ID would need to be extracted differently
			// For now, we'll just use the user ID from API key
		}

		// Parse request body as JSON if possible
		var requestBodyJSON *models.JSONB
		if len(requestBody) > 0 {
			var jsonData map[string]interface{}
			if err := json.Unmarshal(requestBody, &jsonData); err == nil {
				jsonb := models.JSONB(jsonData)
				requestBodyJSON = &jsonb
			}
		}

		// Create audit log
		auditLog := &models.AuditLog{
			APIKeyID:       apiKeyID,
			JWTTokenID:     jwtTokenID,
			UserID:         userID,
			Endpoint:       c.FullPath(),
			Method:         c.Request.Method,
			StatusCode:     c.Writer.Status(),
			RequestBody:    requestBodyJSON,
			ResponseTimeMs: responseTime,
			IPAddress:      c.ClientIP(),
		}

		// Save audit log asynchronously
		go func() {
			_ = auditLogRepo.Create(auditLog)
		}()
	}
}
