package encoders

import (
	"io"
)

// Encoder is the interface that defines a content encoder.
type Encoder interface {
	Encode(handler ContentEngine, writer io.Writer, content interface{}) error
}

// Decoder is the interface that defines a content decoder.
type Decoder interface {
	Decode(handler ContentEngine, reader io.Reader, contentReceiver interface{}) error
}
