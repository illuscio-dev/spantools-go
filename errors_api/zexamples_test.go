package errors_api

import (
	"fmt"
	"spantools/encoders"
	"strings"
)
import "spantools/mimetype"

// EXAMPLES ##########

// Lets convert an error thrown from NewContentEngine.Decode into a
// RequestValidationError as if we are an endpoint handler decoding a request.
func ExampleSpanErrorType_New() {
	// Set up the engine doing our decoding
	engine, _ := encoders.NewContentEngine(false)

	// This data cannot be serialized to a map via json
	data := "YOU'LL NEVER DECODE ME, BATMAN! HAHAHAHAHAHA"
	receiver := make(map[string]string)
	reader := strings.NewReader(data)

	err := engine.Decode(mimetype.JSON, receiver, reader)
	if err != nil {
		// Make a new RequestValidationError
		error := RequestValidationError.New(
			"error reading request content: "+err.Error(),
			nil,
			err,
		)

		// Print the span error
		fmt.Println(error.Error())

		// Do something with the error
		// ...
	}

	fmt.Println()
	// Output:
	// RequestValidationError (1003) - error reading request content: decode err: json decode error [pos 1]: read map - expect char '{' but got char 'Y'
}
