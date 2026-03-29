package httpx

import "github.com/gin-gonic/gin"

const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "requestId"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type ResponseEnvelope struct {
	Data      any            `json:"data"`
	Error     *ErrorResponse `json:"error"`
	RequestID string         `json:"requestId"`
}

func BuildData(c *gin.Context, data any) ResponseEnvelope {
	return ResponseEnvelope{
		Data:      data,
		Error:     nil,
		RequestID: GetRequestID(c),
	}
}

func BuildErrorEnvelope(c *gin.Context, code string, message string, details any) ResponseEnvelope {
	return ResponseEnvelope{
		Data: nil,
		Error: &ErrorResponse{
			Code:    code,
			Message: message,
			Details: details,
		},
		RequestID: GetRequestID(c),
	}
}

func WriteData(c *gin.Context, status int, data any) {
	c.JSON(status, BuildData(c, data))
}

func AbortData(c *gin.Context, status int, data any) {
	c.AbortWithStatusJSON(status, BuildData(c, data))
}

func WriteError(c *gin.Context, status int, code string, message string, details any) {
	c.JSON(status, BuildErrorEnvelope(c, code, message, details))
}

func AbortError(c *gin.Context, status int, code string, message string, details any) {
	c.AbortWithStatusJSON(status, BuildErrorEnvelope(c, code, message, details))
}

func SetRequestID(c *gin.Context, requestID string) {
	c.Set(RequestIDKey, requestID)
}

func GetRequestID(c *gin.Context) string {
	requestID, ok := c.Get(RequestIDKey)
	if !ok {
		return ""
	}

	requestIDText, ok := requestID.(string)
	if !ok {
		return ""
	}

	return requestIDText
}
