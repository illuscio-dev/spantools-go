package encoders

import (
	"bytes"
	"fmt"
	"golang.org/x/xerrors"
	"io"
)

// TODO: Add ability to register custom formatting functions for named types.

// Handled encoding to / decoding from text/plain
type textEncoder struct{}

func (handler *textEncoder) Encode(
	engine ContentEngine, writer io.Writer, content interface{},
) error {
	contentString := fmt.Sprint(content)
	_, err := io.WriteString(writer, contentString)

	return err
}

func (handler *textEncoder) Decode(
	engine ContentEngine, reader io.Reader, contentReceiver interface{},
) error {
	stringPointer, ok := contentReceiver.(*string)
	if !ok {
		return xerrors.New(
			"content receiver must be a string pointer to receive a string.",
		)
	}

	buffer := new(bytes.Buffer)
	if _, err := buffer.ReadFrom(reader); err != nil {
		return err
	}

	*stringPointer = buffer.String()

	return nil
}
