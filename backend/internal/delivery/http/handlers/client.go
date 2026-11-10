package handlers

import "github.com/gin-gonic/gin"

// ClientInfo holds client metadata extracted from an incoming HTTP request.
type ClientInfo struct {
	IPAddress string
	UserAgent string
}

// GetClientInfo extracts both the client IP address and User-Agent from the Gin context.
func GetClientInfo(c *gin.Context) ClientInfo {
	return ClientInfo{
		IPAddress: GetClientIP(c),
		UserAgent: GetUserAgent(c),
	}
}

// GetClientIP extracts the client IP address from the Gin context.
func GetClientIP(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return c.ClientIP()
}

// GetUserAgent extracts the User-Agent string from the Gin context.
func GetUserAgent(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return c.Request.UserAgent()
}
