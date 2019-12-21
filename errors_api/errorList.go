package errors_api

// Base Error. Used when generic error is returned by route handler.
var APIError = SpanErrorType{
	Name:     "APIError",
	HttpCode: 501,
	ApiCode:  1000,
}

// Route does not implement HTTP method (GET, POST, PUT, etc.)
var InvalidMethodError = SpanErrorType{
	Name:     "InvalidMethodError",
	HttpCode: 405,
	ApiCode:  1001,
}

// No media to return.
var NothingToReturnError = SpanErrorType{
	Name:     "NothingToReturnError",
	HttpCode: 400,
	ApiCode:  1002,
}

// Error Occurred when Reading / validating Request Data.
var RequestValidationError = SpanErrorType{
	Name:     "RequestValidationError",
	HttpCode: 400,
	ApiCode:  1003,
}

// Request Exceeds API limit.
var APILimitError = SpanErrorType{
	Name:     "APILimitError",
	HttpCode: 400,
	ApiCode:  1004,
}

// Error occurred when writing Response.
var ResponseValidationError = SpanErrorType{
	Name:     "ResponseValidationError",
	HttpCode: 400,
	ApiCode:  1005,
}

// Sent back when the server framework raises an error that SpanServer does not handle.
// This type SHOULD NOT be invoked by app logic.
var ServerError = SpanErrorType{
	Name:     "Server Error",
	HttpCode: -1,
	ApiCode:  1006,
}

var ErrorList = [7]*SpanErrorType{
	&APIError,
	&InvalidMethodError,
	&NothingToReturnError,
	&RequestValidationError,
	&APILimitError,
	&ResponseValidationError,
	&ServerError,
}

func makeDefaultErrorCodeIndex() map[int]*SpanErrorType {
	index := make(map[int]*SpanErrorType)
	for _, errorType := range ErrorList {
		index[errorType.ApiCode] = errorType
	}
	return index
}

var ErrorTypeCodeIndex = makeDefaultErrorCodeIndex()
