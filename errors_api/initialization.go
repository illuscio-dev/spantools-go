package errors_api

import (
	uuid "github.com/satori/go.uuid"
	"golang.org/x/xerrors"
	"runtime/debug"
	"spantools/encoders"
	"spantools/mimetype"
	"strconv"
	"strings"
)

// Returns a new span error to be returned by the route handler or panicked.
func NewSpanError(
	errorType *SpanErrorType,
	message string,
	errorData map[string]interface{},
	source error,
) *SpanError {
	spanError := SpanError{
		ErrorType:   errorType,
		Message:     message,
		ID:          uuid.NewV4(),
		ErrorData:   errorData,
		sourceErr:   source,
		sourceStack: debug.Stack(),
		frame:       xerrors.Caller(0),
	}
	return &spanError
}

// Creates a new error that is immediately passed to a panic. Expected to be recovered
// by the SpanError middleware. Allows for errors_api to be generated from anywhere
// inside the route handle without need to explicitly pass them up a chain of nested
// function returns.
func PanicSpanError(
	errorType *SpanErrorType,
	message string,
	errorData map[string]interface{},
	source error,
) {
	spanError := NewSpanError(errorType, message, errorData, source)
	panic(spanError)
}

type headerFetcher interface {
	Get(key string) string
}

// ErrorFromHeaders generates error object from headers of HTTP response. If a spanError
// object can be  made from the header data, a pointer to it is returned. If a spanError
// code is detected in the headers, but the header data is malformed and cannot be
// loaded, then hasError is returned as True, and a description of the parsing issue is
// returned in err.
//
// If the headers do not contain an error and hasError will be False, spanError will
// be returned as a nil pointer, and err will specify that no error was found.
func ErrorFromHeaders(
	headers headerFetcher,
	dataEngine encoders.ContentEngine,
	errorTypeCodeIndex map[int]*SpanErrorType,
) (spanError *SpanError, hasError bool, err error) {

	// If there is no error code, then there is no error
	errorCodeStr := headers.Get("error-code")
	if errorCodeStr == "" {
		return nil, false, xerrors.New("no error in headers")
	}

	// If the error code is not an int, then there is no error
	errorCode, err := strconv.Atoi(errorCodeStr)
	if err != nil {
		return nil, false, xerrors.New("error-code not int")
	}

	if errorTypeCodeIndex == nil {
		return nil,
			true,
			xerrors.New("no error index provided")
	}
	errorType, ok := errorTypeCodeIndex[errorCode]
	if !ok {
		return nil,
			true,
			xerrors.New("no known error for code " + errorCodeStr)
	}

	errorMessage := headers.Get("error-message")
	errorIDStr := headers.Get("error-id")

	errorID, err := uuid.FromString(errorIDStr)
	if err != nil {
		return nil,
			true,
			xerrors.New("error ID is not valid UUID")
	}

	errorData := make(map[string]interface{})
	if errorDataStr := headers.Get("error-data"); errorDataStr != "" {
		stringReader := strings.NewReader(errorDataStr)
		err := dataEngine.Decode(mimetype.JSON, errorData, stringReader)
		if err != nil {
			return nil,
				true,
				xerrors.New("error data could not be parsed as JSON")
		}
	}

	spanError = NewSpanError(
		errorType, errorMessage, errorData, nil,
	)
	spanError.ID = errorID

	return spanError, true, nil
}
