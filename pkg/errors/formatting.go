package errors

import "fmt"

// UserError formats user-facing error messages consistently
func UserError(format string, args ...interface{}) {
	fmt.Printf("❌ %s\n", fmt.Sprintf(format, args...))
}

// UserWarning formats user-facing warning messages consistently
func UserWarning(format string, args ...interface{}) {
	fmt.Printf("⚠️  %s\n", fmt.Sprintf(format, args...))
}

// UserInfo formats user-facing info messages consistently
func UserInfo(format string, args ...interface{}) {
	fmt.Printf("ℹ️  %s\n", fmt.Sprintf(format, args...))
}

// ConnectionWarning formats connection-related warnings
func ConnectionWarning(format string, args ...interface{}) {
	fmt.Printf("Warning: %s\n", fmt.Sprintf(format, args...))
}

// DatabaseError formats database-related errors with context
func DatabaseError(operation string, err error) {
	fmt.Printf("❌ Database %s failed: %v\n", operation, err)
}

// APIError formats API-related errors with helpful context
func APIError(service string, err error) {
	fmt.Printf("❌ %s error: %v\n", service, err)
}