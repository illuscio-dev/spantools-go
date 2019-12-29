package tests

//revive:disable:import-shadowing

import (
	"bou.ke/monkey"
	"bytes"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"
	"io"
	"reflect"
	"spantools/encoding"
	"spantools/mimetype"
	"spantools/spantypes"
	"testing"
)

func TestBSONListRoundTrip(test *testing.T) {
	engine := createEngine(test)

	data := []Name{
		{
			First: "Harry",
			Last:  "Potter",
		},
		{
			First: "Hermione",
			Last:  "Granger",
		},
		{
			First: "Ron",
			Last:  "Weasley",
		},
	}

	buffer := &bytes.Buffer{}

	err := engine.Encode(mimetype.BSON, &data, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("DUMPED:", buffer.String())

	loaded := make([]Name, 0)
	err = engine.Decode(mimetype.BSON, &loaded, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("ELEMENT 1:", loaded[0].First)
	assert.Equal(test, data, loaded)
}

func TestBSONListRoundTripPointers(test *testing.T) {
	engine := createEngine(test)

	data := []*Name{
		{
			First: "Harry",
			Last:  "Potter",
		},
		{
			First: "Hermione",
			Last:  "Granger",
		},
		{
			First: "Ron",
			Last:  "Weasley",
		},
	}

	buffer := &bytes.Buffer{}

	err := engine.Encode(mimetype.BSON, &data, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("DUMPED:", buffer.String())

	loaded := make([]*Name, 0)
	err = engine.Decode(mimetype.BSON, &loaded, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("ELEMENT 1:", loaded[0].First)
	assert.Equal(test, data, loaded)
}

func TestBSONRawToBSON(test *testing.T) {
	engine := createEngine(test)

	type Receiver struct {
		Data spantypes.BinData
	}

	buffer := bytes.NewBuffer(make([]byte, 0))
	_, err := io.WriteString(buffer, "Test Data.")
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

	buffer = bytes.NewBuffer(make([]byte, 0))
	err = engine.Encode(mimetype.BSON, &rawDoc, buffer)
	if err != nil {
		test.Error(err)
	}

	loaded := Receiver{}
	err = engine.Decode(mimetype.BSON, &loaded, buffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(test, spantypes.BinData(binData), loaded.Data)
}

func TestUUIDToBSON(test *testing.T) {
	engine := createEngine(test)

	type Receiver struct {
		Data uuid.UUID
	}

	data := Receiver{Data: uuid.NewV4()}

	buffer := bytes.Buffer{}
	engine.Encode(mimetype.BSON, &data, &buffer)

	test.Logf("Dumped: %s", buffer.String())

	loaded := Receiver{}
	engine.Decode(mimetype.BSON, &loaded, &buffer)

	assert.Equal(test, data.Data, loaded.Data)
}

type BinReceiver struct {
	Data spantypes.BinData
}

func setupBinData(
	test *testing.T, engine encoding.ContentEngine,
) (dumpedObject *BinReceiver, contentBuffer *bytes.Buffer) {
	buffer := new(bytes.Buffer)
	_, err := io.WriteString(buffer, "Test Data.")
	if err != nil {
		test.Error(err)
	}

	binData := buffer.Bytes()
	data := &BinReceiver{Data: spantypes.BinData(binData)}

	buffer = new(bytes.Buffer)
	err = engine.Encode(mimetype.BSON, &data, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Logf("Dumped: %s", buffer.String())
	return data, buffer
}

func TestBinBlobToBSON(test *testing.T) {
	engine := createEngine(test)

	dumpedObj, contentBuffer := setupBinData(test, engine)

	loaded := BinReceiver{}
	err := engine.Decode(mimetype.BSON, &loaded, contentBuffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(test, dumpedObj.Data, loaded.Data)
}

func TestUnmarshalToBinDataUnknownError(test *testing.T) {
	engine := createEngine(test)

	_, contentBuffer := setupBinData(test, engine)

	mockReadBinary := func(src []byte) (subtype byte, bin []byte, rem []byte, ok bool) {
		return 0x0, make([]byte, 0), make([]byte, 0), false
	}

	monkey.Patch(
		bsoncore.ReadBinary,
		mockReadBinary,
	)

	loaded := new(BinReceiver)
	err := engine.Decode(mimetype.BSON, loaded, contentBuffer)
	assert.EqualError(
		test, err, "decode err: unknown error decoding spantools.BinData",
	)
}

func TestUnmarshalToBinDataWrongSubtype(test *testing.T) {
	engine := createEngine(test)

	_, contentBuffer := setupBinData(test, engine)

	mockReadBinary := func(src []byte) (subtype byte, bin []byte, rem []byte, ok bool) {
		return 0x5, make([]byte, 0), make([]byte, 0), true
	}

	monkey.Patch(
		bsoncore.ReadBinary,
		mockReadBinary,
	)

	loaded := new(BinReceiver)
	err := engine.Decode(mimetype.BSON, loaded, contentBuffer)
	assert.EqualError(
		test,
		err,
		"decode err: spantools.BinData field is not bson subtype 0x0",
	)
}

func TestErrorDecodingUUID(test *testing.T) {
	engine := createEngine(test)
	buffer := &bytes.Buffer{}

	type TestData struct {
		ID uuid.UUID
	}

	data := map[string]string{"ID": "not an ID"}

	err := engine.Encode(mimetype.BSON, data, buffer)
	if err != nil {
		test.Error(err)
	}

	receiver := &TestData{}
	err = engine.Decode(mimetype.BSON, receiver, buffer)
	assert.EqualError(
		test,
		err,
		"decode err: uuid: UUID must be exactly 16 bytes long, got 0 bytes",
	)
}

func TestErrorMarshall(test *testing.T) {
	engine := createEngine(test)
	buffer := &bytes.Buffer{}

	data := "I am a string"

	err := engine.Encode(mimetype.BSON, data, buffer)
	assert.EqualError(
		test,
		err,
		"encode err: WriteString can only write while positioned on a "+
			"Element or Value but is positioned on a TopLevel",
	)
}

func TestBSONListMustBePointer(test *testing.T) {
	engine := createEngine(test)

	data := []*Name{
		{
			First: "Harry",
			Last:  "Potter",
		},
	}

	buffer := &bytes.Buffer{}

	err := engine.Encode(mimetype.BSON, &data, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("DUMPED:", buffer.String())

	loaded := make([]*Name, 0)
	err = engine.Decode(mimetype.BSON, loaded, buffer)

	assert.EqualError(test, err, "decode err: slice receiver must be pointer")
}

func TestBSONListEncodeErrorWithElement(test *testing.T) {
	engine := createEngine(test)

	data := []string{"I am a string"}
	buffer := &bytes.Buffer{}

	err := engine.Encode(mimetype.BSON, data, buffer)
	assert.EqualError(
		test, err, "encode err: WriteString can only write while "+
			"positioned on a Element or Value but is positioned on a TopLevel",
	)
}

func TestBSONListDecodeErrorWithElement(test *testing.T) {
	engine := createEngine(test)

	data := []*Name{
		{
			First: "Harry",
			Last:  "Potter",
		},
	}

	buffer := &bytes.Buffer{}

	err := engine.Encode(mimetype.BSON, &data, buffer)
	if err != nil {
		test.Error(err)
	}

	type NotName struct {
		First int
		Last  int
	}

	test.Log("DUMPED:", buffer.String())

	loaded := make([]*NotName, 0)
	err = engine.Decode(mimetype.BSON, &loaded, buffer)

	assert.EqualError(
		test, err, "decode err: cannot decode string into an integer type",
	)
}

func TestBSONListEncodeErrorWritingSeparator(test *testing.T) {
	engine := createEngine(test)

	data := []*Name{
		{
			First: "Harry",
			Last:  "Potter",
		},
		{
			First: "Harry",
			Last:  "Potter",
		},
	}

	buffer := &bytes.Buffer{}

	mockBufferWrite := func(buff *bytes.Buffer, data []byte) (int, error) {
		if string(data) == encoding.BsonListSepString {
			return 0, xerrors.New("mock error")
		}

		return len(data), nil
	}

	defer monkey.UnpatchAll()
	monkey.PatchInstanceMethod(
		reflect.TypeOf(&bytes.Buffer{}),
		"Write",
		mockBufferWrite,
	)

	err := engine.Encode(mimetype.BSON, data, buffer)
	assert.EqualError(
		test, err, ": : ",
	)
}
