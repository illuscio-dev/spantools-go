package errors_api

// Base Error. Used when generic error is returned by route handler.
var APIError = NewSpanErrorType(
	"APIError",
	1000,
	502,
)

// Route does not implement HTTP method (GET, POST, PUT, etc.)
var InvalidMethodError = NewSpanErrorType(
	"InvalidMethodError",
	1001,
	405,
)

// No media to return.
var NothingToReturnError = NewSpanErrorType(
	"NothingToReturnError",
	1002,
	400,
)

// Error Occurred when Reading / validating Request Data.
var RequestValidationError = NewSpanErrorType(
	"RequestValidationError",
	1003,
	400,
)

// Request Exceeds API limit.
var APILimitError = NewSpanErrorType(
	"APILimitError",
	1004,
	400,
)

// Error occurred when writing Response.
var ResponseValidationError = NewSpanErrorType(
	"ResponseValidationError",
	1005,
	400,
)

// Sent back when the server framework raises an error that SpanServer does not handle.
// This type SHOULD NOT be invoked by app logic.
var ServerError = NewSpanErrorType(
	"ServerError",
	1006,
	-1,
)

// List of default SpanError definitions.
var ErrorList = [7]*SpanErrorType{
	APIError,
	InvalidMethodError,
	NothingToReturnError,
	RequestValidationError,
	APILimitError,
	ResponseValidationError,
	ServerError,
}

// Used to make ErrorTypeCodeIndex.
func makeDefaultErrorCodeIndex() map[int]*SpanErrorType {
	index := make(map[int]*SpanErrorType)
	for _, errorType := range ErrorList {
		index[errorType.apiCode] = errorType
	}
	return index
}

// ApiCode:*ErrorType indexing of default errors.
var ErrorTypeCodeIndex = makeDefaultErrorCodeIndex()
