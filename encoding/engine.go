package encoding

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

// Type helpers
type encoderMapping map[mimetype.MimeType]Encoder
type decoderMapping map[mimetype.MimeType]Decoder

/*
ContentEngine details the contract for a content encoding engine. The goal of the
content engine is to allow a common decoding and encoding methodology for any
supported mimetype, allowing easy support for client-requested payload encodings, and
a shared interface for different types of services to add support for various
encoding types.
*/
type ContentEngine interface {
	// Registers an encoder for a given mimetype.
	SetEncoder(mimeType mimetype.MimeType, encoder Encoder)

	// Registers a decoder for a given mimetype.
	SetDecoder(mimeType mimetype.MimeType, decoder Decoder)

	// Returns true if the engine has a registered encoder for the mimetype.
	HandlesEncode(mimeType mimetype.MimeType) bool

	// Returns true if the engine has a registered decoder for the mimetype.
	HandlesDecode(mimeType mimetype.MimeType) bool

	// Returns true if the engine has a registered encoder AND decoder for the mimetype.
	Handles(mimeType mimetype.MimeType) bool

	// Whether the engine will attempt to encode / decode unknown mimetypes.
	SniffType() bool

	// Decode mimeType content from reader using the decoder for mimeType. Decoded
	// content is stored in contentReceiver.
	Decode(
		mimeType mimetype.MimeType,
		contentReceiver interface{},
		reader io.Reader,
	) error

	// Encode content as mimetype using registered mimeType to writer.
	Encode(
		mimeType mimetype.MimeType,
		content interface{},
		writer io.Writer,
	) error
}

/*
SpanEngine is the default implementation of the ContentEngine interface.
Implementation is done through an Interface so that the Engine can be extended
through type wrapping.

Instantiation

Use NewContentEngine() to create a new SpanEngine.

Default Mimetypes

• plain/text

• application/json

• application/bson

Object encoding/decoders have been selected to be extensible, and SpanEngine exposes
functions to let you add custom type handlers to each.

Default JSON Extensions

SpanEngine uses the codec library to encode/decode json
(https://godoc.org/github.com/ugorji/go/codec), which allows the definition of
extensions. SpanEngine ships with the following types handled:

• UUIDs from "github.com/satori/go.uuid"

• Binary blob data represented as []byte or *[]bytes are represented as a hex string.
To signal that this conversion should take place, you must use the named type
BinData in the "spantypes" package of this module.

• BSON primitive.Binary data will be decoded as a string for 0x3 subtype (UUID) and a
hex string for 0x0 subtype (arbitrary binary data). Other subtypes are not currently
supported and will panic.

• BSON raw is converted to a map and THEN encoded to a json object.

Additional json extensions can be registered through the AddJSONExtensions() by passing
a slice of JSONExtensionOpts objects.

Default BSON Codecs

SpanEngine handles the encoding and decoding of Bson data through the official bson
driver (https://godoc.org/go.mongodb.org/mongo-driver).

See information on defining a bson codec
here: https://godoc.org/go.mongodb.org/mongo-driver/bson/bsoncodec

The following type extensions ship with SpanEngine:

• primitive.Binary of subtype 0x3 can be decoded to / encoded from UUID objects from
"github.com/satori/go.uuid".

• primitive.Binary of subtype 0x0 can be decoded to / encoded from the BinData named
type of []byte in the "spantypes" module.

Default Text/Plain Returns

When encoding to plaintext, format.Sprint is used on the passed object, so any type
can be sent and represented as text.

Type Sniffing

If created with "sniffMimeType" set to true, when decoding SpanEngine will attempt
to use each decoder until one does not return an error or panic. Because decoders are
internally stored in a map, the order of these attempts is not guaranteed to be
consistent.

Panics

If an encoder or decoder panics during execution, that panic is caught and returned as
an error.
*/
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
	// Engine to pass to Encoder.Encoder() and Decoder.Decode() methods.
	passedEngine ContentEngine
}

// Change the engine passed into Encoder.Encode() and decoder.Decode()
func (engine *SpanEngine) SetPassedEngine(newEngine ContentEngine) {
	engine.passedEngine = newEngine
}

// Register an encoder for a given mimeType
func (engine *SpanEngine) SetEncoder(mimeType mimetype.MimeType, encoder Encoder) {
	engine.encoders[mimeType] = encoder
}

// Register a decoder for a given mimeType
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

// Whether SpanEngine will attempt to decode UNKNOWN content.
func (engine *SpanEngine) SniffType() bool {
	return engine.sniffMimeType
}

// Whether the SpanEngine has a registered encoder for mimeType.
func (engine *SpanEngine) HandlesEncode(mimeType mimetype.MimeType) bool {
	_, ok := engine.encoders[mimeType]
	return ok
}

// Whether the SpanEngine has a registered decoder for mimeType.
func (engine *SpanEngine) HandlesDecode(mimeType mimetype.MimeType) bool {
	// Register the decoder.
	_, ok := engine.decoders[mimeType]
	return ok
}

// Whether the SpanEngine has a registered decoder AND encoder for mimeType.
func (engine *SpanEngine) Handles(mimeType mimetype.MimeType) bool {
	return engine.HandlesEncode(mimeType) && engine.HandlesDecode(mimeType)
}

// Select what engine to pass into the encoder / decoder in case we are extending
// the engine type.
func (engine *SpanEngine) getEngine() (passEngine ContentEngine) {
	if engine.passedEngine != nil {
		passEngine = engine.passedEngine
	} else {
		passEngine = engine
	}

	return passEngine
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

	passEngine := engine.getEngine()
	err = encoder.Encode(passEngine, writer, content)
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

	passEngine := engine.getEngine()
	err = decoder.Decode(passEngine, reader, contentReceiver)

	return err
}

// Attempts to decode content with all registered decoders until one succeeds or all
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

func (engine *SpanEngine) JSONHandle() *codec.JsonHandle {
	return engine.jsonHandle
}

// Returns the internal bsoncodec.BSONRegistry used by the bson encoder/decoder.
func (engine *SpanEngine) BSONRegistry() *bsoncodec.Registry {
	return engine.bsonRegistry
}

// Adds JSON extensions to handle.
func (engine *SpanEngine) AddJSONExtensions(extensions []*JSONExtensionOpts) error {
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

// Adds BSON codecs to engine for use when encoding/decoding bson data.
func (engine *SpanEngine) AddBSONCodecs(codecs []*BsonCodecOpts) error {
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

func NewContentEngine(allowSniff bool) (*SpanEngine, error) {
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

	// Add the encoding.
	engine.SetEncoder(mimetype.JSON, &jsonEncoder{})
	engine.SetEncoder(mimetype.BSON, &bsonEncoder{})
	engine.SetEncoder(mimetype.TEXT, &textEncoder{})

	// Add the default decoders.
	engine.SetDecoder(mimetype.JSON, &jsonEncoder{})
	engine.SetDecoder(mimetype.BSON, &bsonEncoder{})
	engine.SetDecoder(mimetype.TEXT, &textEncoder{})

	// Add the default json extensions to the engine.
	if err := engine.AddJSONExtensions(defaultJSONExtensions); err != nil {
		err = xerrors.Errorf("error adding default json extensions: %w", err)
		return nil, err
	}

	// Add the default bson codecs to the engine.
	if err := engine.AddBSONCodecs(defaultBsonCodecs); err != nil {
		err = xerrors.Errorf("error adding default bson codecs: %w", err)
		return nil, err
	}

	return engine, nil
}
