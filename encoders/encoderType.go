package encoders

import (
	"io"
)

// Interface for defining a content encoder.
type Encoder interface {
	// To be implemented by content encoder. Implementation is expected to write content
	// to writer. The content engine which is calling Encode is made available through
	// engine, allowing encoders to access engine-level settings.
	Encode(engine ContentEngine, writer io.Writer, content interface{}) error
}

// Interface for defining a content decoder.
type Decoder interface {
	// To be implemented by content decoder. Implementation is expected to read content
	// from reader and unmarshal it into contentReceiver. The content engine which is
	// calling Decode is made available through engine, allowing decoders to access
	// engine-level settings.
	Decode(handler ContentEngine, reader io.Reader, contentReceiver interface{}) error
}
