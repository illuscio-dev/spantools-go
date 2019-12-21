package tests

import (
	"bou.ke/monkey"
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/ugorji/go/codec"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
	"io"
	"reflect"
	"spantools/encoders"
	"spantools/mimetype"
	"testing"
)

type Name struct {
	First string
	Last  string
}

type PanickyEncoder struct{}

func (encoder *PanickyEncoder) Encode(
	handler encoders.ContentEngine, writer io.Writer, content interface{},
) error {
	panic(xerrors.New("encode panicked"))
}

func (encoder *PanickyEncoder) Decode(
	handler encoders.ContentEngine, reader io.Reader, contentReceiver interface{},
) error {
	panic(xerrors.New("decode panicked"))
}

func createEngine(test *testing.T) encoders.ContentEngine {
	engine, err := encoders.NewContentEngine(true)
	if err != nil {
		test.Error(err)
	}
	return engine
}

func TestCreateEngineDefault(test *testing.T) {
	engine, err := encoders.NewContentEngine(false)

	assert.Nil(test, err)
	assert.NotNil(test, engine)

	_ = engine.(encoders.ContentEngine)
	_ = engine.(*encoders.SpanEngine)

	// Test that all the defaults registered appropriately.
	assert.Equal(test, true, engine.Handles(mimetype.JSON))
	assert.Equal(test, true, engine.Handles(mimetype.BSON))
	assert.Equal(test, true, engine.Handles(mimetype.TEXT))

	assert.Equal(test, false, engine.Handles(mimetype.MimeType("text/csv")))

	assert.Equal(test, false, engine.SniffType())
}

// Generic function for round-tripping a basic name object for a given mimeType
func RoundTripName(
	test *testing.T, mimeTypeEncode mimetype.MimeType, mimeTypeDecode mimetype.MimeType,
) *Name {
	engine := createEngine(test)

	testName := Name{
		First: "Harry",
		Last:  "Potter",
	}

	buffer := bytes.Buffer{}

	err := engine.Encode(mimeTypeEncode, testName, &buffer)
	if err != nil {
		test.Error(err)
	}

	loaded := Name{}
	err = engine.Decode(mimeTypeDecode, &loaded, &buffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(test, testName, loaded)
	assert.Equal(test, "Harry", loaded.First)
	assert.Equal(test, "Potter", loaded.Last)

	return &loaded
}

func TestJsonBasicRoundTrip(test *testing.T) {
	RoundTripName(test, mimetype.JSON, mimetype.JSON)
}

func TestBsonBasicRoundTrip(test *testing.T) {
	RoundTripName(test, mimetype.BSON, mimetype.BSON)
}

func TestUnknownObjectBasicRoundTrip(test *testing.T) {
	RoundTripName(test, mimetype.UNKNOWN, mimetype.UNKNOWN)
}

func TestJSONToUnknownRoundTrip(test *testing.T) {
	RoundTripName(test, mimetype.JSON, mimetype.UNKNOWN)
}

func TestBSONToUnknownRoundTrip(test *testing.T) {
	RoundTripName(test, mimetype.BSON, mimetype.UNKNOWN)
}

func TestTextRoundTrip(test *testing.T) {
	engine, err := encoders.NewContentEngine(false)
	if err != nil {
		test.Error(test)
	}

	stringPayload := "Test String."
	buffer := bytes.Buffer{}

	err = engine.Encode(mimetype.TEXT, stringPayload, &buffer)
	if err != nil {
		test.Error(err)
	}

	loaded := ""
	err = engine.Decode(mimetype.TEXT, &loaded, &buffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(test, stringPayload, loaded)
}

func TestTextRoundUnknown(test *testing.T) {
	engine, err := encoders.NewContentEngine(true)
	if err != nil {
		test.Error(test)
	}

	stringPayload := "Test String."
	buffer := bytes.Buffer{}

	err = engine.Encode(mimetype.UNKNOWN, stringPayload, &buffer)
	if err != nil {
		test.Error(err)
	}

	loaded := ""
	err = engine.Decode(mimetype.UNKNOWN, &loaded, &buffer)
	if err != nil {
		test.Error(err)
	}

	assert.Equal(test, stringPayload, loaded)
}

func TestNoDecoderError(test *testing.T) {
	engine := createEngine(test)
	buffer := &bytes.Buffer{}
	receiver := make(map[string]interface{})

	err := engine.Decode("text/csv", receiver, buffer)

	assert.EqualError(test, err, "no decoder for text/csv")
}

func TestNoEncoderError(test *testing.T) {
	engine := createEngine(test)
	buffer := &bytes.Buffer{}
	data := make(map[string]interface{})

	err := engine.Encode("text/csv", data, buffer)

	assert.EqualError(test, err, "no encoder for text/csv")
}

func TestEncodePanicsError(test *testing.T) {
	engine := createEngine(test)
	buffer := &bytes.Buffer{}

	decoder := &PanickyEncoder{}
	engine.SetEncoder("text/csv", decoder)

	data := make(map[string]interface{})
	err := engine.Encode("text/csv", data, buffer)

	assert.EqualError(
		test, err, "encode err: panic during encode: encode panicked",
	)
}

func TestDecoderPanicsError(test *testing.T) {
	engine := createEngine(test)
	buffer := &bytes.Buffer{}

	decoder := &PanickyEncoder{}
	engine.SetDecoder("text/csv", decoder)

	data := make(map[string]interface{})
	err := engine.Decode("text/csv", data, buffer)

	assert.EqualError(
		test, err, "decode err: panic during decode: decode panicked",
	)
}

func TestNoSniffError(test *testing.T) {
	engine, err := encoders.NewContentEngine(false)
	if err != nil {
		test.Error(err)
	}

	buffer := &bytes.Buffer{}
	receiver := make(map[string]interface{})

	err = engine.Decode(mimetype.UNKNOWN, receiver, buffer)
	assert.EqualError(
		test, err, "mimetype is unknown and sniffing is disabled",
	)
}

func TestSniffFailsError(test *testing.T) {
	assert := assert.New(test)
	engine := createEngine(test)
	buffer := &bytes.Buffer{}

	type TestSubData struct {
		Field string
	}

	type TestData struct {
		SubData string
	}

	data := make(map[string]interface{})
	subdata := make(map[string]interface{})
	subdata["Field"] = 10
	data["SubData"] = subdata

	err := engine.Encode(mimetype.JSON, data, buffer)
	if err != nil {
		test.Error(err)
	}

	test.Log("DUMPED:", buffer.String())

	receiver := &TestData{}

	err = engine.Decode(mimetype.UNKNOWN, receiver, buffer)
	assert.Contains(
		err.Error(),
		"content receiver must be a string pointer to receive a string",
	)
	assert.Contains(
		err.Error(),
		"unexpected EOF",
	)
	assert.Contains(
		err.Error(),
		"read json delimiter",
	)
}

func TestSniffErrorReadingBytes(test *testing.T) {
	mockReadFrom := func(buffer *bytes.Buffer, reader io.Reader) (int64, error) {
		return 0, xerrors.New("mock reader error")
	}

	defer monkey.UnpatchAll()
	monkey.PatchInstanceMethod(
		reflect.TypeOf(&bytes.Buffer{}),
		"ReadFrom",
		mockReadFrom,
	)
	engine := createEngine(test)

	buffer := &bytes.Buffer{}
	receiver := make(map[string]interface{})

	err := engine.Decode(mimetype.UNKNOWN, receiver, buffer)
	assert.EqualError(
		test, err, "error reading contentBytes: mock reader error",
	)
}

func TestErrorAddindJsonHandle(test *testing.T) {
	mockSetInterfaceExt := func(
		handle *codec.JsonHandle, rt reflect.Type, tag uint64, ext codec.InterfaceExt,
	) error {
		return xerrors.New("mock error")
	}

	defer monkey.UnpatchAll()
	monkey.PatchInstanceMethod(
		reflect.TypeOf(&codec.JsonHandle{}),
		"SetInterfaceExt",
		mockSetInterfaceExt,
	)

	_, err := encoders.NewContentEngine(false)
	assert.EqualError(
		test,
		err,
		"error adding default json extensions: error adding json extension"+
			" to content engine: mock error",
	)
}

func TestErrorAddingBsonCodec(test *testing.T) {
	// Because the bson codec add only returns an error from adding the json handler,
	// we can just mock that.
	mockSetInterfaceExt := func(
		handle *codec.JsonHandle, rt reflect.Type, tag uint64, ext codec.InterfaceExt,
	) error {
		if rt == reflect.TypeOf(bson.Raw{}) {
			return xerrors.New("mock error")
		} else {
			return nil
		}
	}

	defer monkey.UnpatchAll()
	monkey.PatchInstanceMethod(
		reflect.TypeOf(&codec.JsonHandle{}),
		"SetInterfaceExt",
		mockSetInterfaceExt,
	)

	_, err := encoders.NewContentEngine(false)
	assert.EqualError(
		test,
		err,
		"error adding default bson codecs: error building bson extension "+
			"for json handle: mock error",
	)
}

type TestCloser struct {
	Buffer *bytes.Buffer
	Closed bool
}

func (closer *TestCloser) Read(p []byte) (n int, err error) {
	return closer.Buffer.Read(p)
}

func (closer *TestCloser) Close() error {
	closer.Closed = true
	return nil
}

func TestClosesReader(test *testing.T) {
	assert := assert.New(test)

	engine := createEngine(test)
	buffer := &bytes.Buffer{}

	name := &Name{
		First: "Harry",
		Last:  "Potter",
	}

	err := engine.Encode(mimetype.JSON, name, buffer)
	if err != nil {
		test.Error(err)
	}

	closer := &TestCloser{
		Buffer: buffer,
	}

	assert.False(closer.Closed)

	loaded := &Name{}
	err = engine.Decode(mimetype.JSON, loaded, closer)
	if err != nil {
		test.Error(err)
	}

	assert.True(closer.Closed)
	assert.Equal(name, loaded)
}
