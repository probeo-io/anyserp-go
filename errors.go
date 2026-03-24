package anyserp

import "fmt"

// AnySerpError represents an error returned by a search provider.
type AnySerpError struct {
	Code     int
	Message  string
	Metadata map[string]interface{}
}

func (e *AnySerpError) Error() string {
	if name, ok := e.Metadata["provider_name"]; ok {
		return fmt.Sprintf("anyserp [%v] %d: %s", name, e.Code, e.Message)
	}
	return fmt.Sprintf("anyserp %d: %s", e.Code, e.Message)
}

// NewAnySerpError creates a new AnySerpError.
func NewAnySerpError(code int, message string, metadata map[string]interface{}) *AnySerpError {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	return &AnySerpError{
		Code:     code,
		Message:  message,
		Metadata: metadata,
	}
}
