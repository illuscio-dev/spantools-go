package errors_api

import (
	"bytes"
	"fmt"
	"github.com/satori/go.uuid"
	"golang.org/x/xerrors"
	"spantools/encoders"
	"spantools/mimetype"
	"strconv"
)

type headerSetter interface {
	Set(key string, value string)
}

type SpanErrorType struct {
	Name     string
	ApiCode  int
	HttpCode int
}

// Allows the error type definition itself to also be a valid error for things like
// testing error equality.
func (errorType *SpanErrorType) Error() string {
	return errorType.Name +
		" (" + strconv.Itoa(errorType.ApiCode) + ")"
}

type SpanError struct {
	ErrorType   *SpanErrorType
	Message     string
	ID          uuid.UUID
	ErrorData   map[string]interface{}
	sourceErr   error
	sourceStack []byte
	frame       xerrors.Frame
}

func (spanError *SpanError) Error() string {
	return spanError.ErrorType.Error() + " - " + spanError.Message
}

func (spanError *SpanError) Unwrap() error {
	// implements xerrors.Wrapper
	return spanError.sourceErr
}

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

func (spanError *SpanError) ToHeader(
	setter headerSetter, dataEngine encoders.ContentEngine,
) error {
	setter.Set("error-name", spanError.ErrorType.Name)
	setter.Set("error-code", strconv.Itoa(spanError.ErrorType.ApiCode))
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
