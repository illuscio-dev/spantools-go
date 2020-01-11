package tests

//revive:disable:import-shadowing reason: Disabled for assert := assert.New(), which is
// the preferred method of using multiple asserts in a test.

import (
	"bou.ke/monkey"
	"bytes"
	"github.com/stretchr/testify/assert"
	"golang.org/x/xerrors"
	"io"
	"reflect"
	"github.com/illuscio-dev/spantools-go/mimetype"
	"testing"
)

func TestPanickedReader(test *testing.T) {
	assert := assert.New(test)
	engine := createEngine(test)

	mockReadFrom := func(buffer *bytes.Buffer, reader io.Reader) (int64, error) {
		return 0, xerrors.New("mock reader error")
	}

	defer monkey.UnpatchAll()
	monkey.PatchInstanceMethod(
		reflect.TypeOf(&bytes.Buffer{}),
		"ReadFrom",
		mockReadFrom,
	)

	var receiver *string

	buffer := &bytes.Buffer{}

	mimeType, err := engine.Decode(mimetype.TEXT, receiver, buffer)
	assert.Zero(mimeType)
	assert.EqualError(err, "decode err: mock reader error")
}
