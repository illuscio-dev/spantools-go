package encoding

import (
	uuid "github.com/satori/go.uuid"
	"github.com/ugorji/go/codec"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/xerrors"
	"io"
	"reflect"
	"github.com/illuscio-dev/spantools-go/spantypes"
)

// JSONExtensionOpts holds options For Json Handle extension to add to the handle on
// server setup.
type JSONExtensionOpts struct {
	ValueType    reflect.Type
	ExtInterface codec.InterfaceExt
}

// defaultJSONExtensions holds all the JSONExtensionOpts to add to the JSONHandle on
// server setup
var defaultJSONExtensions = []*JSONExtensionOpts{
	{
		ValueType:    reflect.TypeOf(primitive.Binary{}),
		ExtInterface: &jsonExtBsonBinary{},
	},
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

	unmarshaled := make(map[string]interface{})

	if len(valueRaw) > 0 {
		err := bson.UnmarshalWithRegistry(
			ext.bsonRegistry, valueRaw, &unmarshaled,
		)
		if err != nil {
			panic(xerrors.Errorf(
				"error while unmarshalling bson for encoding: %w", err,
			))
		}
	}

	return unmarshaled
}

func (ext *jsonExtBsonRaw) UpdateExt(dest interface{}, value interface{}) {
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
