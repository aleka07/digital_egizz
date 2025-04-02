// Package api contains the API definitions for Digital Egiz
package api

import (
	"encoding/json"
	"net/http"
)

// ResponseWriter is a utility for writing consistent API responses
type ResponseWriter struct {
	Writer http.ResponseWriter
}

// DigitalTwin represents a basic digital twin entity
type DigitalTwin struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Properties  map[string]interface{} `json:"properties"`
}

// ApiResponse is a standard response structure for the API
type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewResponseWriter creates a new ResponseWriter with the given http.ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{Writer: w}
}

// SendJSON sends a JSON response with the given status code and data
func (rw *ResponseWriter) SendJSON(statusCode int, data interface{}) {
	rw.Writer.Header().Set("Content-Type", "application/json")
	rw.Writer.WriteHeader(statusCode)
	json.NewEncoder(rw.Writer).Encode(data)
}

// SendSuccess sends a successful API response
func (rw *ResponseWriter) SendSuccess(statusCode int, message string, data interface{}) {
	resp := ApiResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	rw.SendJSON(statusCode, resp)
}

// SendError sends an error API response
func (rw *ResponseWriter) SendError(statusCode int, message string) {
	resp := ApiResponse{
		Success: false,
		Error:   message,
	}
	rw.SendJSON(statusCode, resp)
}

/*
API Endpoints Design

The following endpoints will be implemented for digital twin operations:

1. Create Digital Twin
   - Path: /api/v1/twins
   - Method: POST
   - Request Body: DigitalTwin without ID
   - Response: Created DigitalTwin with ID
   - Status Codes: 201 Created, 400 Bad Request, 500 Internal Server Error

2. Get Digital Twin
   - Path: /api/v1/twins/{id}
   - Method: GET
   - Response: DigitalTwin
   - Status Codes: 200 OK, 404 Not Found, 500 Internal Server Error

3. Update Digital Twin
   - Path: /api/v1/twins/{id}
   - Method: PUT
   - Request Body: DigitalTwin
   - Response: Updated DigitalTwin
   - Status Codes: 200 OK, 400 Bad Request, 404 Not Found, 500 Internal Server Error

4. Delete Digital Twin
   - Path: /api/v1/twins/{id}
   - Method: DELETE
   - Response: Success message
   - Status Codes: 200 OK, 404 Not Found, 500 Internal Server Error

5. List Digital Twins
   - Path: /api/v1/twins
   - Method: GET
   - Query Parameters: page, limit, filter
   - Response: Array of DigitalTwin
   - Status Codes: 200 OK, 500 Internal Server Error
*/
