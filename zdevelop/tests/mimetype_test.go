package tests

//revive:disable:import-shadowing reason: Disabled for assert := assert.New(), which is
// the preferred method of using multiple asserts in a test.

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"github.com/illuscio-dev/spantools-go/mimetype"
	"testing"
)

func ParameterizeFromString(
	test *testing.T, testStrings []string, mimeTypeExpected mimetype.MimeType,
) {
	for _, mimeTypeString := range testStrings {
		mimeTypeExtracted := mimetype.FromString(mimeTypeString)
		assert.Equal(test, mimeTypeExpected, mimeTypeExtracted)
	}
}

func ParameterizeFromHeader(
	test *testing.T, testStrings []string, mimeTypeExpected mimetype.MimeType,
) {
	for _, mimeTypeString := range testStrings {
		req := http.Request{
			Header: make(http.Header),
		}
		req.Header.Set("Content-Type", mimeTypeString)
		mimeTypeExtracted := mimetype.FromHeader(req.Header)
		assert.Equal(test, mimeTypeExpected, mimeTypeExtracted)
	}
}

func TestFromJson(test *testing.T) {
	stringValues := []string{
		"json",
		"JSON",
		"x-json",
		"application/json",
		"application/JSON",
		"application/x-json",
		"application/X-JSON",
	}

	testFromString := func(subTest *testing.T) {
		ParameterizeFromString(test, stringValues, mimetype.JSON)
	}
	testFromHeader := func(subTest *testing.T) {
		ParameterizeFromHeader(test, stringValues, mimetype.JSON)
	}

	test.Run("JSON From String", testFromString)
	test.Run("JSON From Header", testFromHeader)

}

func TestFromBson(test *testing.T) {
	stringValues := []string{
		"bson",
		"BSON",
		"x-bson",
		"application/bson",
		"application/BSON",
		"application/x-bson",
		"application/X-BSON",
	}
	testFromString := func(subTest *testing.T) {
		ParameterizeFromString(test, stringValues, mimetype.BSON)
	}
	testFromHeader := func(subTest *testing.T) {
		ParameterizeFromHeader(test, stringValues, mimetype.BSON)
	}

	test.Run("BSON From String", testFromString)
	test.Run("BSON From Header", testFromHeader)
}

func TestFromYaml(test *testing.T) {
	stringValues := []string{
		"yaml",
		"YAML",
		"x-yaml",
		"application/yaml",
		"application/YAML",
		"application/x-yaml",
		"application/X-YAML",
	}
	testFromString := func(subTest *testing.T) {
		ParameterizeFromString(test, stringValues, mimetype.YAML)
	}
	testFromHeader := func(subTest *testing.T) {
		ParameterizeFromHeader(test, stringValues, mimetype.YAML)
	}

	test.Run("YAML From String", testFromString)
	test.Run("YAML From Header", testFromHeader)
}

func TestFromText(test *testing.T) {
	stringValues := []string{
		"text",
		"TEXT",
		"text/plain",
		"TEXT/plain",
	}
	testFromString := func(subTest *testing.T) {
		ParameterizeFromString(test, stringValues, mimetype.TEXT)
	}
	testFromHeader := func(subTest *testing.T) {
		ParameterizeFromHeader(test, stringValues, mimetype.TEXT)
	}

	test.Run("TEXT From String", testFromString)
	test.Run("TEXT From Header", testFromHeader)
}

func TestFromUnknown(test *testing.T) {
	stringValues := []string{""}

	testFromString := func(subTest *testing.T) {
		ParameterizeFromString(test, stringValues, mimetype.UNKNOWN)
	}
	testFromHeader := func(subTest *testing.T) {
		ParameterizeFromHeader(test, stringValues, mimetype.UNKNOWN)
	}

	test.Run("UNKNOWN From String", testFromString)
	test.Run("UNKNOWN From Header", testFromHeader)
}

func TestFromStringOther(test *testing.T) {
	stringValues := []string{"text/csv", "TEXT/CSV", "text/CSV"}
	expected := mimetype.MimeType("text/csv")

	testFromString := func(subTest *testing.T) {
		ParameterizeFromString(test, stringValues, expected)
	}
	testFromHeader := func(subTest *testing.T) {
		ParameterizeFromHeader(test, stringValues, expected)
	}

	test.Run("Other From String", testFromString)
	test.Run("Other From Header", testFromHeader)
}
