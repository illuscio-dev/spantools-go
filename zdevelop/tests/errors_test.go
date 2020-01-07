package tests

//revive:disable:import-shadowing reason: Disabled for assert := assert.New(), which is
// the preferred method of using multiple asserts in a test.

import (
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/xerrors"
	"net/http"
	"reflect"
	"spantools/encoding"
	"spantools/spanerrors"
	"testing"
)

// Creates a consistent test error for multiple tests
func createTestError() *spanerrors.SpanError {
	sourceErr := xerrors.New("some source error")

	spanErr := spanerrors.ResponseValidationError.New(
		"test message",
		map[string]interface{}{"key": "value"},
		sourceErr,
	)
	return spanErr
}

// Helper function to verify the error created by createTestError() in multiple
// tests.
func verifyError(test *testing.T, spanErr *spanerrors.SpanError) {
	assert := assert.New(test)

	assert.Equal(spanerrors.ResponseValidationError, spanErr.SpanErrorType)
	assert.NotEqual(uuid.Nil, spanErr.Id)
	assert.Equal("test message", spanErr.Message)
	assert.Equal(map[string]interface{}{"key": "value"}, spanErr.ErrorData)
	assert.Error(xerrors.New("some source error"), spanErr.Unwrap())
}

// Sets up a test error, test request with headers, and content engine for running
// tests where we need to dump to or pull from headers.
func setupHeadersTest(
	test *testing.T,
) (*spanerrors.SpanError, *http.Request, encoding.ContentEngine) {
	testReq := http.Request{
		Header: make(http.Header),
	}
	return createTestError(), &testReq, createEngine(test)
}

func TestNewSpanError(test *testing.T) {
	assert := assert.New(test)

	spanErr := createTestError()
	verifyError(test, spanErr)

	assert.Equal("ResponseValidationError", spanErr.Name())
	assert.Equal(1005, spanErr.ApiCode())
	assert.Equal(400, spanErr.HttpCode())

	assert.True(spanErr.IsType(spanerrors.ResponseValidationError))
	assert.False(spanErr.IsType(spanerrors.RequestValidationError))
}

func TestPanicSpanError(test *testing.T) {
	// Used this to verify that we have panicked
	assert := assert.New(test)

	panicked := false

	// Since the defer here executes at the end of the function, we need to wrap it
	// in another function so we can verify that the defer took place.
	func() {
		defer func() {
			recovered := recover()
			spanErr := recovered.(*spanerrors.SpanError)

			verifyError(test, spanErr)
			assert.Equal("ResponseValidationError", spanErr.Name())
			assert.Equal(1005, spanErr.ApiCode())
			assert.Equal(400, spanErr.HttpCode())

			assert.True(spanErr.IsType(spanerrors.ResponseValidationError))
			assert.False(spanErr.IsType(spanerrors.RequestValidationError))

			panicked = true
		}()

		sourceErr := xerrors.New("some source error")

		// This should cause a panic.
		spanerrors.ResponseValidationError.Panic(
			"test message",
			map[string]interface{}{"key": "value"},
			sourceErr,
		)
	}()

	assert.True(panicked)
}

func TestWithHttpCodeType(test *testing.T) {
	assert := assert.New(test)

	assert.Equal(spanerrors.ServerError.HttpCode(), -1)
	spanErrType := spanerrors.ServerError.WithHttpCode(500)
	assert.Equal(spanErrType.HttpCode(), 500)

	spanErr := spanErrType.New("some message", nil, nil)

	assert.True(spanErr.IsType(spanerrors.ServerError))
	assert.False(spanErr.IsType(spanerrors.RequestValidationError))
}

func TestSpanErrorMessage(test *testing.T) {
	spanErr := createTestError()

	assert.Equal(
		test, "ResponseValidationError (1005) - test message", spanErr.Error(),
	)
}

func TestSpanLogMessage(test *testing.T) {
	sourceErr := xerrors.New("some source error")

	spanErr := spanerrors.ResponseValidationError.New(
		"test message",
		nil,
		sourceErr,
	)

	logMessage := spanErr.LogMessage()

	assert.Contains(
		test,
		logMessage,
		"MESSAGE: ResponseValidationError (1005) - test message",
	)
	assert.Contains(
		test, logMessage, "ORIGINAL: some source error",
	)
	assert.Contains(
		test, logMessage, "PANIC STACK:",
	)
	assert.Contains(
		test, logMessage, "runtime/debug.Stack(",
	)
}

func TestToHeaders(test *testing.T) {
	assert := assert.New(test)

	spanErr, testReq, engine := setupHeadersTest(test)

	err := spanErr.ToHeader(testReq.Header, engine)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(
		"ResponseValidationError", testReq.Header.Get("error-name"),
	)
	assert.Equal("1005", testReq.Header.Get("error-code"))
	assert.Equal("test message", testReq.Header.Get("error-message"))
	assert.NotEqual("", testReq.Header.Get("error-id"))
	assert.Equal("{\"key\":\"value\"}", testReq.Header.Get("error-data"))
}

func TestFromHeaders(test *testing.T) {
	assert := assert.New(test)

	spanErr, testReq, engine := setupHeadersTest(test)

	err := spanErr.ToHeader(testReq.Header, engine)
	if err != nil {
		test.Error(err)
	}

	errLoaded, hasErr, err := spanerrors.ErrorFromHeaders(
		testReq.Header, engine, spanerrors.ErrorTypeCodeIndex,
	)
	if err != nil {
		test.Error(err)
	}

	assert.True(hasErr)
	assert.Equal(spanErr.Error(), errLoaded.Error())
	assert.Equal(spanErr.Id, errLoaded.Id)
	assert.Equal(spanErr.ErrorData, errLoaded.ErrorData)
}

type badType string

type jsonExtBadType struct{}

func (ext *jsonExtBadType) ConvertExt(value interface{}) interface{} {
	panic(xerrors.New("Whoops"))
}

func (ext *jsonExtBadType) UpdateExt(dest interface{}, value interface{}) {
	panic(xerrors.New("Whoops"))
}

// Tests that
func TestErrorDumpingData(test *testing.T) {
	spanErr, testReq, engine := setupHeadersTest(test)
	spanEngine := engine.(*encoding.SpanEngine)

	badTypeOpts := encoding.JSONExtensionOpts{
		ValueType:    reflect.TypeOf(badType("")),
		ExtInterface: &jsonExtBadType{},
	}
	err := spanEngine.AddJSONExtensions([]*encoding.JSONExtensionOpts{&badTypeOpts})
	if err != nil {
		test.Error(err)
	}

	spanErr.ErrorData["key2"] = badType("Bad Type")

	dumpErr := spanErr.ToHeader(testReq.Header, engine)

	assert.EqualError(test, dumpErr, "encode err: json encode error: Whoops")
}

func TestNoErrorInHeaders(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)
	testReq := http.Request{
		Header: make(http.Header),
	}

	spanErr, hasErr, err := spanerrors.ErrorFromHeaders(
		testReq.Header, engine, spanerrors.ErrorTypeCodeIndex,
	)

	assert.Nil(spanErr)
	assert.False(hasErr)
	assert.EqualError(err, "no error in headers")
}

func TestErrorCodeNotInt(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)
	testReq := http.Request{
		Header: make(http.Header),
	}
	testReq.Header.Set("error-code", "not an int")

	spanErr, hasErr, err := spanerrors.ErrorFromHeaders(
		testReq.Header, engine, spanerrors.ErrorTypeCodeIndex,
	)

	assert.Nil(spanErr)
	assert.False(hasErr)
	assert.EqualError(err, "error-code not int")
}

func TestErrorCodeNoKnown(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)
	testReq := http.Request{
		Header: make(http.Header),
	}
	testReq.Header.Set("error-code", "9999")

	spanErr, hasErr, err := spanerrors.ErrorFromHeaders(
		testReq.Header, engine, spanerrors.ErrorTypeCodeIndex,
	)

	assert.Nil(spanErr)
	assert.True(hasErr)
	assert.EqualError(err, "no known error for code 9999")
}

func TestErrorBadID(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)
	testReq := http.Request{
		Header: make(http.Header),
	}
	testReq.Header.Set("error-code", "1005")
	testReq.Header.Set("error-id", "not a uuid")

	spanErr, hasErr, err := spanerrors.ErrorFromHeaders(
		testReq.Header, engine, spanerrors.ErrorTypeCodeIndex,
	)

	assert.Nil(spanErr)
	assert.True(hasErr)
	assert.EqualError(err, "error Id is not valid UUID")
}

func TestErrorBadData(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)
	testReq := http.Request{
		Header: make(http.Header),
	}
	testReq.Header.Set("error-code", "1005")
	testReq.Header.Set("error-id", uuid.NewV4().String())
	testReq.Header.Set("error-data", "not valid json object")

	spanErr, hasErr, err := spanerrors.ErrorFromHeaders(
		testReq.Header, engine, spanerrors.ErrorTypeCodeIndex,
	)

	assert.Nil(spanErr)
	assert.True(hasErr)
	assert.EqualError(err, "error data could not be parsed as JSON")
}

func TestErrorNoIndex(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)
	testReq := http.Request{
		Header: make(http.Header),
	}
	testReq.Header.Set("error-code", "1005")
	testReq.Header.Set("error-id", uuid.NewV4().String())

	spanErr, hasErr, err := spanerrors.ErrorFromHeaders(
		testReq.Header, engine, nil,
	)

	assert.Nil(spanErr)
	assert.True(hasErr)
	assert.EqualError(err, "no error index provided")
}

func TestCustomErrorFromHeader(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)
	testReq := http.Request{
		Header: make(http.Header),
	}

	CustomErrorType := spanerrors.NewSpanErrorType(
		"CustomError",
		2001,
		400,
	)

	CustomErrorIndex := make(map[int]*spanerrors.SpanErrorType)
	for key, value := range spanerrors.ErrorTypeCodeIndex {
		CustomErrorIndex[key] = value
	}
	CustomErrorIndex[CustomErrorType.ApiCode()] = CustomErrorType

	testReq.Header.Set("error-code", "2001")
	testReq.Header.Set("error-id", uuid.NewV4().String())

	spanErr, hasErr, err := spanerrors.ErrorFromHeaders(
		testReq.Header, engine, CustomErrorIndex,
	)

	assert.NotNil(spanErr)
	assert.True(hasErr)
	assert.Nil(err)
	assert.EqualError(spanErr.SpanErrorType, CustomErrorType.Error())
}
