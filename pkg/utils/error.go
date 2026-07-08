package utils

// CustomError carries the HTTP status code alongside the message, so callers
// don't need to pass status and message as separate WriteError arguments.
type CustomError struct {
	StatusCode int
	Message    string
}

func (e *CustomError) Error() string {
	return e.Message
}

func NewCustomError(statusCode int, message string) error {
	return &CustomError{StatusCode: statusCode, Message: message}
}
