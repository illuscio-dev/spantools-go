package encoders

import (
	"bytes"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"golang.org/x/xerrors"
	"io"
	"reflect"
	"spantools/mimetype"
)
import "github.com/ugorji/go/codec"

type encoderMapping map[mimetype.MimeType]Encoder
type decoderMapping map[mimetype.MimeType]Decoder

// ContentEngine is the interface for objects that hold multiple content encoders and
// decoders, so that arbitrary content types can be handled through a common interface.
type ContentEngine interface {
	SetEncoder(mimeType mimetype.MimeType, encoder Encoder)
	SetDecoder(mimeType mimetype.MimeType, decoder Decoder)
	HandlesEncode(mimeType mimetype.MimeType) bool
	HandlesDecode(mimeType mimetype.MimeType) bool
	Handles(mimeType mimetype.MimeType) bool
	SniffType() bool
	Decode(
		mimeType mimetype.MimeType,
		contentReceiver interface{},
		reader io.Reader,
	) error
	Encode(
		mimeType mimetype.MimeType,
		content interface{},
		writer io.Writer,
	) error
}

// SpanEngine is the default implementation of the ContentEngine interface.
// Implementation is done through an Interface so that the Engine can be extended
// through type wrapping.
type SpanEngine struct {
	// MimeType:Encoder mapping
	encoders encoderMapping
	// MimeType:Decoder mapping
	decoders decoderMapping
	// List of all registered decoders. Used for sniffing mimetype.
	decoderList []Decoder
	// Whether to attempt decoding when no explicit mimetype is known.
	sniffMimeType bool

	// JSON handle for default JSON encoder
	jsonHandle *codec.JsonHandle
	// BSON registry for default BSON encoder
	bsonRegistry *bsoncodec.Registry
	// BSON codecs
	bsonCodecs []*BsonCodecOpts
}

func (engine *SpanEngine) SetEncoder(mimeType mimetype.MimeType, encoder Encoder) {
	engine.encoders[mimeType] = encoder
}

func (engine *SpanEngine) SetDecoder(mimeType mimetype.MimeType, decoder Decoder) {
	// Set the encoder.
	engine.decoders[mimeType] = decoder

	// Cache a list of all the decoders we can use when mimetype sniffing. Because of
	// this SNIFF ORDER IS NOT GUARANTEED.
	engine.decoderList = make([]Decoder, len(engine.decoders))
	index := 0

	for _, decoder := range engine.decoders {
		engine.decoderList[index] = decoder
		index++
	}
}

func (engine *SpanEngine) SniffType() bool {
	return engine.sniffMimeType
}

func (engine *SpanEngine) HandlesEncode(mimeType mimetype.MimeType) bool {
	_, ok := engine.encoders[mimeType]
	return ok
}

func (engine *SpanEngine) HandlesDecode(mimeType mimetype.MimeType) bool {
	// Register the decoder.
	_, ok := engine.decoders[mimeType]
	return ok
}

func (engine *SpanEngine) Handles(mimeType mimetype.MimeType) bool {
	return engine.HandlesEncode(mimeType) && engine.HandlesDecode(mimeType)
}

// Uses a decoder while catching panics to return as errors
func (engine *SpanEngine) safeEncode(
	encoder Encoder, writer io.Writer, content interface{},
) (err error) {
	defer func() {
		recovered := recover()
		if recovered != nil {
			err = xerrors.Errorf("panic during encode: %w", recovered)
		}
	}()

	err = encoder.Encode(engine, writer, content)
	return err
}

// Uses a decoder while catching panics to return as errors
func (engine *SpanEngine) safeDecode(
	decoder Decoder, reader io.Reader, contentReceiver interface{},
) (err error) {
	defer func() {
		recovered := recover()
		if recovered != nil {
			err = xerrors.Errorf("panic during decode: %w", recovered)
		}
	}()

	err = decoder.Decode(engine, reader, contentReceiver)

	return err
}

// attempts to decode content with all registered decoders until one succeeds or all
// fail.
func (engine *SpanEngine) sniffContent(
	mimeType mimetype.MimeType,
	contentReceiver interface{},
	reader io.Reader,
) error {
	// We need to read the content multiple times, so lets load the bytes into a var.
	// This will cause a slight performance hit, which is why this is a separate process
	// from loading a KNOWN mimetype.
	contentBuffer := bytes.NewBuffer(make([]byte, 0))
	if _, err := contentBuffer.ReadFrom(reader); err != nil {
		return xerrors.Errorf("error reading contentBytes: %w", err)
	}

	var decoderErr error
	var decoded bool

	for _, decoder := range engine.decoderList {
		// Make a buffer for this attempt, otherwise we'll run out of bytes.
		thisReader := bytes.NewBuffer(contentBuffer.Bytes())
		thisErr := engine.safeDecode(decoder, thisReader, contentReceiver)

		if thisErr != nil {
			if decoderErr == nil {
				decoderErr = thisErr
			} else {
				decoderErr = xerrors.Errorf(
					"decoding error: %w after: %w", thisErr, decoderErr,
				)
			}
		} else {
			decoded = true
			break
		}
	}

	if decoded {
		decoderErr = nil
	}

	return decoderErr
}

// Picks the mimetype for encoding / decoding objects when source or target mimetype is
// unknown.
func pickContentMimeType(
	mimeType mimetype.MimeType, content interface{}, encoding bool,
) mimetype.MimeType {
	if mimeType == mimetype.UNKNOWN {
		var useType mimetype.MimeType

		switch content.(type) {
		case string:
			useType = mimetype.TEXT
		case *string:
			useType = mimetype.TEXT
		default:
			useType = mimetype.JSON
		}

		// If we are decoding, we only want to force a text decoding if the receiver is
		// a string.
		if encoding || useType == mimetype.TEXT {
			mimeType = useType
		}
	}
	return mimeType
}

func (engine *SpanEngine) Decode(
	mimeType mimetype.MimeType,
	contentReceiver interface{},
	reader io.Reader,
) error {
	mimeType = pickContentMimeType(mimeType, contentReceiver, false)

	// Close the reader if it's a closer.
	if readCloser, ok := reader.(io.ReadCloser); ok {
		defer func() {
			_ = readCloser.Close()
		}()
	}

	// If we want to sniff
	if mimeType == mimetype.UNKNOWN {
		if !engine.SniffType() {
			return xerrors.New("mimetype is unknown and sniffing is disabled")
		}
		return engine.sniffContent(mimeType, contentReceiver, reader)
	}

	decoder, ok := engine.decoders[mimeType]
	if !ok {
		return xerrors.New("no decoder for " + string(mimeType))
	}

	err := engine.safeDecode(decoder, reader, contentReceiver)
	if err != nil {
		return xerrors.Errorf("decode err: %w", err)
	}

	return nil
}

func (engine *SpanEngine) Encode(
	mimeType mimetype.MimeType,
	content interface{},
	writer io.Writer,
) error {
	mimeType = pickContentMimeType(mimeType, content, true)

	encoder, ok := engine.encoders[mimeType]
	if !ok {
		return xerrors.New("no encoder for " + string(mimeType))
	}

	err := engine.safeEncode(encoder, writer, content)
	if err != nil {
		return xerrors.Errorf(
			"encode err: %w", err,
		)
	}
	return nil
}

// Adds JSON extensions to handle
func (engine *SpanEngine) AddJsonExtensions(extensions []*JsonExtensionOpts) error {
	for _, extOpts := range extensions {
		err := engine.jsonHandle.SetInterfaceExt(
			extOpts.ValueType, 1, extOpts.ExtInterface,
		)
		if err != nil {
			return xerrors.Errorf(
				"error adding json extension to content engine: %w", err,
			)
		}
	}
	return nil
}

func (engine *SpanEngine) AddBsonCodecs(codecs []*BsonCodecOpts) error {
	// Store these codecs for later in case more are added by the end user and we need
	// to declare a new engine.
	engine.bsonCodecs = append(engine.bsonCodecs, codecs...)

	builder := bsoncodec.NewRegistryBuilder()
	bsoncodec.DefaultValueEncoders{}.RegisterDefaultEncoders(builder)
	bsoncodec.DefaultValueDecoders{}.RegisterDefaultDecoders(builder)

	for _, codecOpts := range codecs {
		builder.RegisterCodec(codecOpts.ValueType, codecOpts.Codec)
	}

	// Build the bson registry.
	engine.bsonRegistry = builder.Build()

	// Now redeclare the json extension for bson raw with this registry so it has access
	// to any additional codecs
	err := engine.jsonHandle.SetInterfaceExt(
		reflect.TypeOf(bson.Raw{}),
		1,
		&jsonExtBsonRaw{engine.bsonRegistry},
	)
	if err != nil {
		return xerrors.Errorf(
			"error building bson extension for json handle: %w", err,
		)
	}

	return nil
}

func NewContentEngine(allowSniff bool) (ContentEngine, error) {
	// Create the json handle.
	jsonHandle := &codec.JsonHandle{}

	// Create the content engine.
	engine := &SpanEngine{
		encoders:      make(encoderMapping),
		decoders:      make(decoderMapping),
		sniffMimeType: allowSniff,
		jsonHandle:    jsonHandle,
		bsonRegistry:  nil,
	}

	// Add the encoders.
	engine.SetEncoder(mimetype.JSON, &jsonEncoder{})
	engine.SetEncoder(mimetype.BSON, &bsonEncoder{})
	engine.SetEncoder(mimetype.TEXT, &textEncoder{})

	// Add the default decoders.
	engine.SetDecoder(mimetype.JSON, &jsonEncoder{})
	engine.SetDecoder(mimetype.BSON, &bsonEncoder{})
	engine.SetDecoder(mimetype.TEXT, &textEncoder{})

	// Add the default json extensions to the engine.
	if err := engine.AddJsonExtensions(defaultJSONExtensions); err != nil {
		err = xerrors.Errorf("error adding default json extensions: %w", err)
		return nil, err
	}

	// Add the default bson codecs to the engine.
	if err := engine.AddBsonCodecs(defaultBsonCodecs); err != nil {
		err = xerrors.Errorf("error adding default bson codecs: %w", err)
		return nil, err
	}

	// Return the engine.
	engineReturn := ContentEngine(engine)
	return engineReturn, nil
}
