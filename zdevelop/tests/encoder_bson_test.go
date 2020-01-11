package tests

//revive:disable:import-shadowing reason: Disabled for assert := assert.New(), which is
// the preferred method of using multiple asserts in a test.

import (
	"bou.ke/monkey"
	"bytes"
	"github.com/illuscio-dev/spantools-go/encoding"
	"github.com/illuscio-dev/spantools-go/mimetype"
	"github.com/illuscio-dev/spantools-go/spantypes"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"
	"io"
	"reflect"
	"testing"
)

func TestBSONListRoundTrip(test *testing.T) {
	assert := assert.New(test)
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

	mimeType, err := engine.Encode(mimetype.BSON, &data, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("DUMPED:", buffer.String())
	assert.Equal(mimetype.BSON, mimeType)

	loaded := make([]Name, 0)
	mimeType, err = engine.Decode(mimetype.BSON, &loaded, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("ELEMENT 1:", loaded[0].First)
	assert.Equal(data, loaded)
	assert.Equal(mimetype.BSON, mimeType)
}

func TestBSONListRoundTripPointers(test *testing.T) {
	assert := assert.New(test)

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

	mimeType, err := engine.Encode(mimetype.BSON, &data, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("DUMPED:", buffer.String())
	assert.Equal(mimetype.BSON, mimeType)

	loaded := make([]*Name, 0)
	mimeType, err = engine.Decode(mimetype.BSON, &loaded, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("ELEMENT 1:", loaded[0].First)
	assert.Equal(mimetype.BSON, mimeType)
	assert.Equal(data, loaded)
}

func TestBSONRawToBSON(test *testing.T) {
	assert := assert.New(test)
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
	mimeType, err := engine.Encode(mimetype.BSON, &rawDoc, buffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(mimetype.BSON, mimeType)

	loaded := Receiver{}
	mimeType, err = engine.Decode(mimetype.BSON, &loaded, buffer)
	if err != nil {
		test.Error(err)
	}
	assert.Equal(mimetype.BSON, mimeType)

	assert.Equal(spantypes.BinData(binData), loaded.Data)
}

func TestUUIDToBSON(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)

	type Receiver struct {
		Data uuid.UUID
	}

	data := Receiver{Data: uuid.NewV4()}

	buffer := bytes.Buffer{}
	mimeType, err := engine.Encode(mimetype.BSON, &data, &buffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(mimetype.BSON, mimeType)

	test.Logf("Dumped: %s", buffer.String())

	loaded := Receiver{}
	mimeType, err = engine.Decode(mimetype.BSON, &loaded, &buffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(mimetype.BSON, mimeType)
	assert.Equal(data.Data, loaded.Data)
}

type BinReceiver struct {
	Data spantypes.BinData
}

func setupBinData(
	test *testing.T, engine encoding.ContentEngine,
) (dumpedObject *BinReceiver, contentBuffer *bytes.Buffer) {
	assert := assert.New(test)

	buffer := new(bytes.Buffer)
	_, err := io.WriteString(buffer, "Test Data.")
	if err != nil {
		test.Error(err)
	}

	binData := buffer.Bytes()
	data := &BinReceiver{Data: spantypes.BinData(binData)}

	buffer = new(bytes.Buffer)
	mimeType, err := engine.Encode(mimetype.BSON, &data, buffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(mimetype.BSON, mimeType)

	test.Logf("Dumped: %s", buffer.String())
	return data, buffer
}

func TestBinBlobToBSON(test *testing.T) {
	engine := createEngine(test)

	dumpedObj, contentBuffer := setupBinData(test, engine)

	loaded := BinReceiver{}
	mimeType, err := engine.Decode(mimetype.BSON, &loaded, contentBuffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(test, mimetype.BSON, mimeType)
	assert.Equal(test, dumpedObj.Data, loaded.Data)
}

func TestUnmarshalToBinDataUnknownError(test *testing.T) {
	assert := assert.New(test)
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
	mimeType, err := engine.Decode(mimetype.BSON, loaded, contentBuffer)
	assert.Zero(mimeType)
	assert.EqualError(
		err, "decode err: unknown error decoding spantools.BinData",
	)
}

func TestUnmarshalToBinDataWrongSubtype(test *testing.T) {
	assert := assert.New(test)
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
	mimeType, err := engine.Decode(mimetype.BSON, loaded, contentBuffer)
	assert.Zero(mimeType)
	assert.EqualError(
		err,
		"decode err: spantools.BinData field is not bson subtype 0x0",
	)
}

func TestErrorDecodingUUID(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)
	buffer := &bytes.Buffer{}

	type TestData struct {
		ID uuid.UUID
	}

	data := map[string]string{"Id": "not an Id"}

	mimeType, err := engine.Encode(mimetype.BSON, data, buffer)
	if err != nil {
		test.Error(err)
	}
	assert.Equal(mimetype.BSON, mimeType)

	receiver := &TestData{}
	mimeType, err = engine.Decode(mimetype.BSON, receiver, buffer)
	assert.Zero(mimeType)
	assert.EqualError(
		err,
		"decode err: uuid: UUID must be exactly 16 bytes long, got 0 bytes",
	)
}

func TestErrorMarshall(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)
	buffer := &bytes.Buffer{}

	data := "I am a string"

	mimeType, err := engine.Encode(mimetype.BSON, data, buffer)
	assert.Zero(mimeType)
	assert.EqualError(
		err,
		"encode err: WriteString can only write while positioned on a "+
			"Element or Value but is positioned on a TopLevel",
	)
}

func TestBSONListMustBePointer(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)

	data := []*Name{
		{
			First: "Harry",
			Last:  "Potter",
		},
	}

	buffer := &bytes.Buffer{}

	mimeType, err := engine.Encode(mimetype.BSON, &data, buffer)
	if err != nil {
		test.Error(err)
	}
	assert.Equal(mimetype.BSON, mimeType)

	test.Log("DUMPED:", buffer.String())

	loaded := make([]*Name, 0)
	mimeType, err = engine.Decode(mimetype.BSON, loaded, buffer)

	assert.Zero(mimeType)
	assert.EqualError(err, "decode err: slice receiver must be pointer")
}

func TestBSONListEncodeErrorWithElement(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)

	data := []string{"I am a string"}
	buffer := &bytes.Buffer{}

	mimeType, err := engine.Encode(mimetype.BSON, data, buffer)
	assert.Zero(mimeType)
	assert.EqualError(
		err, "encode err: WriteString can only write while "+
			"positioned on a Element or Value but is positioned on a TopLevel",
	)
}

func TestBSONListDecodeErrorWithElement(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)

	data := []*Name{
		{
			First: "Harry",
			Last:  "Potter",
		},
	}

	buffer := &bytes.Buffer{}

	mimeType, err := engine.Encode(mimetype.BSON, &data, buffer)
	if err != nil {
		test.Error(err)
	}
	assert.Equal(mimetype.BSON, mimeType)

	type NotName struct {
		First int
		Last  int
	}

	test.Log("DUMPED:", buffer.String())

	loaded := make([]*NotName, 0)
	mimeType, err = engine.Decode(mimetype.BSON, &loaded, buffer)
	assert.Zero(mimeType)
	assert.EqualError(
		err, "decode err: cannot decode string into an integer type",
	)
}

func TestBSONListEncodeErrorWritingSeparator(test *testing.T) {
	assert := assert.New(test)

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

	mimeType, err := engine.Encode(mimetype.BSON, data, buffer)
	assert.Zero(mimeType)
	assert.EqualError(
		err, ": : ",
	)
}
