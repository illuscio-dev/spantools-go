package tests

//revive:disable:import-shadowing reason: Disabled for assert := assert.New(), which is
// the preferred method of using multiple asserts in a test.

import (
	"bou.ke/monkey"
	"bytes"
	"encoding/hex"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/xerrors"
	"io"
	"spantools/mimetype"
	"spantools/spantypes"
	"testing"
)

func TestJsonListRoundTrip(test *testing.T) {
	engine := createEngine(test)

	data := []*Name{
		{
			First: "Harry",
			Last:  "Potter",
		},
		{
			First: "Ron",
			Last:  "Weasley",
		},
	}

	buffer := &bytes.Buffer{}

	err := engine.Encode(mimetype.JSON, &data, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("DUMPED:", buffer.String())

	loaded := make([]*Name, 0)
	err = engine.Decode(mimetype.JSON, &loaded, buffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(test, data, loaded)
}

func TestBsonUUIDToJson(test *testing.T) {
	engine := createEngine(test)

	uuidValue := uuid.NewV4()
	bsonUUID := primitive.Binary{Subtype: 0x3, Data: uuidValue.Bytes()}

	type Receiver struct {
		Id uuid.UUID
	}

	data := bson.M{"Id": bsonUUID}

	buffer := bytes.Buffer{}
	err := engine.Encode(mimetype.JSON, &data, &buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("Dumped: ", buffer.String())

	loaded := Receiver{}
	err = engine.Decode(mimetype.JSON, &loaded, &buffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(test, uuidValue, loaded.Id)
}

func TestBinBlobToJson(test *testing.T) {
	engine := createEngine(test)

	type Receiver struct {
		Data spantypes.BinData
	}

	buffer := bytes.Buffer{}
	_, err := io.WriteString(&buffer, "Test Data.")
	if err != nil {
		test.Error(err)
	}

	binData := spantypes.BinData(buffer.Bytes())
	data := map[string]interface{}{"Data": binData}

	buffer = bytes.Buffer{}
	if err := engine.Encode(mimetype.JSON, &data, &buffer); err != nil {
		test.Error(err)
	}

	test.Logf("Dumped: %s", buffer.String())

	loaded := Receiver{}
	if err := engine.Decode(mimetype.JSON, &loaded, &buffer); err != nil {
		test.Error(err)
	}

	assert.Equal(test, binData, loaded.Data)
}

func TestBinBlobBSONToJson(test *testing.T) {
	engine := createEngine(test)

	type Receiver struct {
		Data spantypes.BinData
	}

	buffer := bytes.Buffer{}
	_, err := io.WriteString(&buffer, "Test Data.")
	if err != nil {
		test.Error(err)
	}

	binData := buffer.Bytes()
	data := bson.M{"Data": primitive.Binary{
		Subtype: 0x0,
		Data:    buffer.Bytes(),
	}}

	buffer = bytes.Buffer{}
	if err := engine.Encode(mimetype.JSON, &data, &buffer); err != nil {
		test.Error(err)
	}

	test.Logf("Dumped: %s", buffer.String())

	loaded := Receiver{}
	if err := engine.Decode(mimetype.JSON, &loaded, &buffer); err != nil {
		test.Error(err)
	}

	assert.Equal(test, spantypes.BinData(binData), loaded.Data)
}

func TestBSONRawToJson(test *testing.T) {
	engine := createEngine(test)

	type Receiver struct {
		Data spantypes.BinData
	}

	buffer := bytes.Buffer{}
	_, err := io.WriteString(&buffer, "Test Data.")
	if err != nil {
		test.Error(err)
	}

	binData := buffer.Bytes()
	data := bson.M{"Data": primitive.Binary{
		Subtype: 0x0,
		Data:    buffer.Bytes(),
	}}

	rawBytes, err := bson.Marshal(&data)
	if err != nil {
		test.Error(err)
	}

	rawDoc := bson.Raw(rawBytes)

	buffer = bytes.Buffer{}
	err = engine.Encode(mimetype.JSON, &rawDoc, &buffer)
	if err != nil {
		test.Error(err)
	}

	loaded := Receiver{}
	err = engine.Decode(mimetype.JSON, &loaded, &buffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(test, spantypes.BinData(binData), loaded.Data)
}

func TestNonHexDecodeError(test *testing.T) {
	assert := assert.New(test)
	engine := createEngine(test)

	data := map[string]interface{}{"Data": "not bin data"}
	buffer := &bytes.Buffer{}

	err := engine.Encode(mimetype.JSON, data, buffer)
	if err != nil {
		test.Error(err)
	}

	type Receiver struct {
		Data spantypes.BinData
	}
	receiver := &Receiver{}

	err = engine.Decode(mimetype.JSON, receiver, buffer)
	assert.EqualError(
		err,
		"decode err: json decode error [pos 22]: could not decode hex: "+
			"encoding/hex: invalid byte: U+006E 'n'",
	)
}

func TestNonHexEncodeErrorLen(test *testing.T) {
	engine := createEngine(test)

	buffer := bytes.Buffer{}
	_, err := io.WriteString(&buffer, "Test Data.")
	if err != nil {
		test.Error(err)
	}

	binData := spantypes.BinData(buffer.Bytes())
	data := map[string]interface{}{"Data": binData}

	mockHexEncode := func(dst []byte, src []byte) int { return 1 }

	monkey.Patch(
		hex.Encode,
		mockHexEncode,
	)
	defer monkey.UnpatchAll()

	contentBuffer := new(bytes.Buffer)
	err = engine.Encode(mimetype.JSON, data, contentBuffer)
	assert.EqualError(
		test,
		err,
		"encode err: json encode error: error encoding BinData to hex",
	)
}

func TestBsonUUIDError(test *testing.T) {
	assert := assert.New(test)
	engine := createEngine(test)

	data := map[string]interface{}{"Data": primitive.Binary{
		Subtype: 0x3,
		Data:    make([]byte, 0),
	}}
	buffer := &bytes.Buffer{}

	err := engine.Encode(mimetype.JSON, data, buffer)
	assert.EqualError(
		err,
		"encode err: json encode error: Error converting bson uuid: uuid: "+
			"UUID must be exactly 16 bytes long, got 0 bytes",
	)
}

func TestBsonBinNotSupportedError(test *testing.T) {
	assert := assert.New(test)
	engine := createEngine(test)

	data := map[string]interface{}{"Data": primitive.Binary{
		Subtype: 0x10,
		Data:    make([]byte, 0),
	}}
	buffer := &bytes.Buffer{}

	err := engine.Encode(mimetype.JSON, data, buffer)
	assert.EqualError(
		err,
		"encode err: json encode error: unsupported Binary BSON format",
	)
}

func TestBsonUnmarshalBSONError(test *testing.T) {
	assert := assert.New(test)
	engine := createEngine(test)

	data := bson.M{"field": "value"}
	dataBytes, err := bson.Marshal(data)
	if err != nil {
		test.Error(dataBytes)
	}
	buffer := &bytes.Buffer{}

	dataRaw := bson.Raw(dataBytes)

	mockUnmarshalWithRegistry := func(
		r *bsoncodec.Registry, data []byte, val interface{},
	) error {
		return xerrors.New("mock error")
	}

	monkey.Patch(
		bson.UnmarshalWithRegistry,
		mockUnmarshalWithRegistry,
	)
	defer monkey.UnpatchAll()

	err = engine.Encode(mimetype.JSON, &dataRaw, buffer)
	assert.EqualError(
		err,
		"encode err: json encode error: error while unmarshalling bson "+
			"for encoding: mock error",
	)
}

func TestUnmarshalToBsonBinError(test *testing.T) {
	assert := assert.New(test)
	engine := createEngine(test)

	data := map[string]interface{}{"Data": hex.EncodeToString(uuid.NewV4().Bytes())}
	buffer := &bytes.Buffer{}

	err := engine.Encode(mimetype.JSON, data, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("DUMPED:", buffer.String())

	type TestData struct {
		Data *primitive.Binary
	}

	receiver := &TestData{}
	err = engine.Decode(mimetype.JSON, receiver, buffer)
	assert.EqualError(
		err, "decode err: json decode error [pos 42]: decoding to bson binary "+
			"field not supported -- use uuid or BinData type as intermediary",
	)
}

func TestUnmarshalToBsonRawError(test *testing.T) {
	assert := assert.New(test)
	engine := createEngine(test)

	data := map[string]interface{}{"Data": map[string]interface{}{"key": "value"}}
	buffer := &bytes.Buffer{}

	err := engine.Encode(mimetype.JSON, data, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("DUMPED:", buffer.String())

	type TestData struct {
		Data *bson.Raw
	}

	receiver := &TestData{}
	err = engine.Decode(mimetype.JSON, receiver, buffer)
	assert.EqualError(
		err, "decode err: json decode error [pos 23]: Decoding to BSON raw "+
			"field not supported",
	)
}
