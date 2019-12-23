package errors_api

import (
	"bytes"
	"fmt"
	"github.com/satori/go.uuid"
	"golang.org/x/xerrors"
	"runtime/debug"
	"spantools/encoders"
	"spantools/mimetype"
	"strconv"
)

// Interface for object that can set header information.
type headerSetter interface {
	Set(key string, value string)
}

/*
Used to define a type of error that a service can return. Think of to define a TYPE of
error that CAN be returned by your ecosystem, but

Each SpanErrorType for a given ecosystem should have a unique Name and APICode.

Codes 1000-1999 are reserved for Spanreeds default error definitions.

Since types are declared as pointers, to protect against accidental mutation of the
error type by other packages, the underlying fields of this struct are private and
accessed through functions. Define new error types using NewSpanErrorType()
*/
type SpanErrorType struct {
	// Unique human-readable name of the error type for the API ecosystem.
	name string

	// Unique number to identify the error type in the API ecosystem.
	apiCode int

	// HTTP code that should be returned when this error type is returned. Set to -1
	// if the http error is determined dynamically.
	httpCode int
}

// Returns a new span error to be returned by the route handler or panicked.
func (errorType *SpanErrorType) New(
	message string,
	errorData map[string]interface{},
	source error,
) *SpanError {
	spanError := SpanError{
		SpanErrorType: errorType,
		Message:       message,
		ID:            uuid.NewV4(),
		ErrorData:     errorData,
		sourceErr:     source,
		sourceStack:   debug.Stack(),
		frame:         xerrors.Caller(0),
	}
	return &spanError
}

/*
Creates a new error that is immediately passed to a panic. Expected to be recovered
by the SpanError middleware. Allows for errors_api to be generated from anywhere
inside the route handle without need to explicitly pass them up a chain of nested
function returns.
*/
func (errorType *SpanErrorType) Panic(
	message string,
	errorData map[string]interface{},
	source error,
) {
	spanError := errorType.New(message, errorData, source)
	panic(spanError)
}

// Unique human-readable name of the error type for the API ecosystem.
func (errorType *SpanErrorType) Name() string {
	return errorType.name
}

// Unique number to identify the error type in the API ecosystem.
func (errorType *SpanErrorType) ApiCode() int {
	return errorType.apiCode
}

// HTTP code that should be returned when this error type is returned. Set to -1
// if the http error is determined dynamically.
func (errorType *SpanErrorType) HttpCode() int {
	return errorType.httpCode
}

// Returns a copy of the error type with the given http code replaced.
func (errorType *SpanErrorType) WithHttpCode(newHttpCode int) *SpanErrorType {
	return &SpanErrorType{
		name:     errorType.name,
		apiCode:  errorType.apiCode,
		httpCode: newHttpCode,
	}
}

// Allows the error type definition itself to also be a valid error for things like
// testing error equality.
func (errorType *SpanErrorType) Error() string {
	return errorType.name +
		" (" + strconv.Itoa(errorType.apiCode) + ")"
}

// Used to return a specific error instance.
type SpanError struct {
	// The type of error we are returning.
	*SpanErrorType

	// A message detailing what caused the error.
	Message string

	// An id for the error being returned.
	ID uuid.UUID

	// A string / any mapping of data related to the error.
	ErrorData map[string]interface{}

	// If this error was returned because of another error, the original error is stored
	// here.
	sourceErr error

	// The debug,Stack() from where this error was instantiated.
	sourceStack []byte

	// The xerrors.Frame from where this error was instantiated.
	frame xerrors.Frame
}

// Returns true if the underlying type of this error is the same as errorType. Some
// errors may have multiple http codes possible, se we can't just compare ErrorType
// field equality directly.
func (spanError *SpanError) IsType(errorType *SpanErrorType) bool {
	return spanError.SpanErrorType.Error() == errorType.Error()
}

// Error string to conform to builtin error interface.
func (spanError *SpanError) Error() string {
	return spanError.SpanErrorType.Error() + " - " + spanError.Message
}

// Implements xerrors.Wrapper interface. Part of how errors are being considered for
// implementation in future GO versions with more traceback support.
func (spanError *SpanError) Unwrap() error {
	// implements xerrors.Wrapper
	return spanError.sourceErr
}

// More verbose error message that includes a debug.Stack() and source error
// information. This is not part of the Error(), Message, or ErrorData by default since
// it may contain sensitive information that is not desirable to return to the client.
func (spanError *SpanError) LogMessage() string {
	loggerMessage := fmt.Sprint(
		// print the error
		"\nMESSAGE: ",
		spanError.Error(),
		"\nORIGINAL: ",
		spanError.sourceErr,
		"\nPANIC STACK:\n",
		string(spanError.sourceStack),
	)
	return loggerMessage
}

// Writes error to an object which implements a Set(key string, value string) method
// like http.Request or http.Response.
func (spanError *SpanError) ToHeader(
	setter headerSetter, dataEngine encoders.ContentEngine,
) error {
	setter.Set("error-name", spanError.name)
	setter.Set("error-code", strconv.Itoa(spanError.apiCode))
	setter.Set("error-message", spanError.Message)
	setter.Set("error-id", spanError.ID.String())

	if spanError.ErrorData != nil {
		dataBytes := bytes.Buffer{}
		err := dataEngine.Encode(mimetype.JSON, spanError.ErrorData, &dataBytes)
		if err != nil {
			return err
		}
		setter.Set("error-data", dataBytes.String())
	}

	return nil
}
