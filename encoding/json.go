package encoding

import (
	"encoding/hex"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"github.com/ugorji/go/codec"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/xerrors"
	"io"
	"reflect"
	"spantools/spantypes"
)

// JsonExtensionOpts holds options For Json Handle extension to add to the handle on
// server setup.
type JsonExtensionOpts struct {
	ValueType    reflect.Type
	ExtInterface codec.InterfaceExt
}

// defaultJSONExtensions holds all the JsonExtensionOpts to add to the JsonHandle on
// server setup
var defaultJSONExtensions = []*JsonExtensionOpts{
	{
		ValueType:    reflect.TypeOf(primitive.Binary{}),
		ExtInterface: &jsonExtBsonBinary{},
	},
	{
		ValueType:    reflect.TypeOf(make(spantypes.BinData, 0)),
		ExtInterface: &jsonExtBinData{},
	},
}

// Json handle extension for sending and receiving binary blob data (ie raw file data)
// in a json field.
type jsonExtBinData struct{}

func (ext *jsonExtBinData) ConvertExt(value interface{}) interface{} {
	valueBin := value.(spantypes.BinData)
	hexValue := hex.EncodeToString(valueBin)

	return hexValue
}

func (ext *jsonExtBinData) UpdateExt(dest interface{}, value interface{}) {
	destVal := dest.(*spantypes.BinData)
	sourceVal := value.(string)

	binData, err := hex.DecodeString(sourceVal)
	if err != nil {
		panic(
			xerrors.Errorf("could not decode hex: %w", err),
		)
	}

	*destVal = binData
}

// Converts BSON binary fields to json. Currently supports Binary blobs and UUIDs.
type jsonExtBsonBinary struct{}

func (ext *jsonExtBsonBinary) ConvertExt(value interface{}) interface{} {
	valueBin := value.(*primitive.Binary)
	if valueBin.Subtype == 0x3 {
		valueUUID, err := uuid.FromBytes(valueBin.Data)
		if err != nil {
			panic(xerrors.Errorf("Error converting bson uuid: %w", err))
		}
		return valueUUID
	}

	if valueBin.Subtype == 0x0 {
		return spantypes.BinData(valueBin.Data)
	}

	panic(xerrors.New("unsupported Binary BSON format"))
}

func (ext *jsonExtBsonBinary) UpdateExt(dest interface{}, value interface{}) {
	panic(
		xerrors.New(
			"decoding to bson binary field not supported -- " +
				"use uuid or BinData type as intermediary",
		),
	)
}

// Converts BSON Raw document to json object.
type jsonExtBsonRaw struct {
	bsonRegistry *bsoncodec.Registry
}

func (ext *jsonExtBsonRaw) ConvertExt(value interface{}) interface{} {
	valueRaw := value.(bson.Raw)
	fmt.Println("valueRaw:", valueRaw)

	unmarshalled := make(map[string]interface{})
	if len(valueRaw) > 0 {
		err := bson.UnmarshalWithRegistry(
			ext.bsonRegistry, valueRaw, &unmarshalled,
		)
		if err != nil {
			panic(xerrors.Errorf(
				"error while unmarshalling bson for encoding: %w", err,
			))
		}
	}

	return unmarshalled
}

func (ext *jsonExtBsonRaw) UpdateExt(dest interface{}, value interface{}) {
	fmt.Println("TEST")
	panic(xerrors.New("Decoding to BSON raw field not supported"))
}

// default JSON encoder for SpanEngine.
type jsonEncoder struct{}

func (encoder *jsonEncoder) Encode(
	engine ContentEngine, writer io.Writer, content interface{},
) error {
	spanEngine := engine.(*SpanEngine)
	jsonEncoder := codec.NewEncoder(writer, spanEngine.jsonHandle)
	return jsonEncoder.Encode(content)
}

func (encoder *jsonEncoder) Decode(
	engine ContentEngine, reader io.Reader, contentReceiver interface{},
) error {
	spanEngine := engine.(*SpanEngine)
	jsonDecoder := codec.NewDecoder(reader, spanEngine.jsonHandle)
	return jsonDecoder.Decode(contentReceiver)
}
