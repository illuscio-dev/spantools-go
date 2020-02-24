package encoding

import (
	"bufio"
	"bytes"
	uuid "github.com/satori/go.uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"golang.org/x/xerrors"
	"io"
	"reflect"
)

// BsonListSepString is a delimiter for top-level bson lists, which bson does not not
// normally support. When multiple documents are being sent in a single payload, the
// unicode SYMBOL FOR RECORD SEPARATOR is used.
// (http://fileformat.info/info/unicode/char/241e/index.htm)
const BsonListSepString = "\u241E"

// BsonListSepBytes is a byte representation of BsonListSepString.
var BsonListSepBytes = []byte(BsonListSepString)

// split function used to separate the bson records.
func splitBsonFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {

	// Return nothing if at end of file and no data passed
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Find the index of a separator
	if i := bytes.Index(data, BsonListSepBytes); i >= 0 {
		return i + 3, data[0:i], nil
	}

	// If at end of file with data return the data
	if atEOF {
		return len(data), data, nil
	}

	return advance, token, err
}

// BSON

// BsonCodecOpts holds options for registering new BSON codecs with SpanEngine.
type BsonCodecOpts struct {
	// Type this codec handles encoding / decoding to.
	ValueType reflect.Type

	// Codec to register for this type.
	Codec bsoncodec.ValueCodec
}

var defaultBsonCodecs = []*BsonCodecOpts{
	{
		ValueType: reflect.TypeOf(uuid.UUID{}),
		Codec:     bsonCodecUUID{},
	},
}

// CODECS

// bsonCodecUUID Handles encoding and decoding of UUID to and from bson.
type bsonCodecUUID struct{}

// Encodes uuid value to bson.
func (codec bsonCodecUUID) EncodeValue(
	encodeCTX bsoncodec.EncodeContext,
	valueWriter bsonrw.ValueWriter,
	value reflect.Value,
) error {
	valueUUID, _ := value.Interface().(uuid.UUID)
	_ = valueWriter.WriteBinaryWithSubtype(valueUUID.Bytes(), 0x3)

	return nil
}

// Decodes uuid value from bson.
func (codec bsonCodecUUID) DecodeValue(
	decodeCTX bsoncodec.DecodeContext,
	valueReader bsonrw.ValueReader,
	value reflect.Value,
) error {
	bytesUUID, _, _ := valueReader.ReadBinary()
	uuidVal, err := uuid.FromBytes(bytesUUID)

	if err != nil {
		return err
	}

	value.Set(reflect.ValueOf(uuidVal))

	return nil
}

// BSON Encoder for writing BSON Data to content.
type bsonEncoder struct{}

func (encoder *bsonEncoder) encodeSingle(
	spanEngine *SpanEngine, writer io.Writer, content interface{},
) error {
	var bodyBSON bson.Raw

	incomingRaw, isRaw := content.(*bson.Raw)

	if !isRaw {
		marshalled, err := bson.MarshalWithRegistry(spanEngine.bsonRegistry, content)
		if err != nil {
			return err
		}
		bodyBSON = marshalled
	} else {
		bodyBSON = *incomingRaw
	}

	_, err := writer.Write(bodyBSON)
	return err
}

// Used to encode multiple bson objects to s single payload.
func (encoder *bsonEncoder) encodeMany(
	spanEngine *SpanEngine, writer io.Writer, content *reflect.Value,
) error {
	// We need to know when we are on the final index so if we hit the last item we
	// know that we don't need to write the separator.
	finalIndex := content.Len() - 1

	for arrayIndex := 0; arrayIndex <= finalIndex; arrayIndex++ {
		// We have to use reflect to grab the items since we don't know what type they
		// are.
		listValue := content.Index(arrayIndex)

		// Encode this single item.
		err := encoder.encodeSingle(spanEngine, writer, listValue.Interface())
		if err != nil {
			return err
		}

		// Write the delimiter if we are not on the final item.
		if arrayIndex != finalIndex {
			_, err = writer.Write(BsonListSepBytes)
			if err != nil {
				return xerrors.Errorf(
					"error writing document separator: %w", err,
				)
			}
		}
	}
	return nil
}

// Detects whether content to encode is a sequence (array or slice)
func (encoder *bsonEncoder) isSequence(value *reflect.Value) bool {
	return value.Kind() == reflect.Slice || value.Kind() == reflect.Array
}

// Encodes bson content
func (encoder *bsonEncoder) Encode(
	engine ContentEngine, writer io.Writer, content interface{},
) (err error) {
	spanEngine := engine.(*SpanEngine)

	// Check if the value is a slice or an array.
	contentValue := reflect.Indirect(reflect.ValueOf(content))
	// Check that it is not a raw document.
	_, isRaw := content.(*bson.Raw)

	if encoder.isSequence(&contentValue) && !isRaw {
		err = encoder.encodeMany(spanEngine, writer, &contentValue)
	} else {
		err = encoder.encodeSingle(spanEngine, writer, content)
	}

	return err
}

// Decodes a single bson document
func (encoder *bsonEncoder) decodeSingle(
	spanEngine *SpanEngine, reader io.Reader, contentReceiver interface{},
) error {
	document, err := bson.NewFromIOReader(reader)
	if err != nil {
		return err
	}

	return bson.UnmarshalWithRegistry(
		spanEngine.bsonRegistry, document, contentReceiver,
	)
}

// Decodes multiple bson elements.
func (encoder *bsonEncoder) decodeMany(
	spanEngine *SpanEngine, reader io.Reader, contentReceiver interface{},
) error {
	slicePointer := reflect.ValueOf(contentReceiver)
	if slicePointer.Kind() != reflect.Ptr {
		return xerrors.New("slice receiver must be pointer")
	}
	sliceValue := slicePointer.Elem()

	// Get the element type for the slice.
	elementType := reflect.TypeOf(contentReceiver).Elem().Elem()
	docScanner := bufio.NewScanner(reader)
	docScanner.Split(splitBsonFunc)

	// Iterate through documents.
	for docScanner.Scan() {
		docBuff := bytes.NewBuffer(docScanner.Bytes())
		newElement := reflect.New(elementType)

		err := encoder.decodeSingle(spanEngine, docBuff, newElement.Interface())
		if err != nil {
			return err
		}

		sliceValue.Set(reflect.Append(sliceValue, newElement.Elem()))
	}

	return nil
}

// Decode bson content
func (encoder *bsonEncoder) Decode(
	engine ContentEngine, reader io.Reader, contentReceiver interface{},
) (err error) {
	spanEngine := engine.(*SpanEngine)
	// Check if the value is a slice or an array.
	receiverValue := reflect.Indirect(reflect.ValueOf(contentReceiver))

	// If the receiver is a slice or array, we need to decode multiple documents.
	if encoder.isSequence(&receiverValue) {
		err = encoder.decodeMany(spanEngine, reader, contentReceiver)
	} else {
		err = encoder.decodeSingle(spanEngine, reader, contentReceiver)
	}

	return err
}
