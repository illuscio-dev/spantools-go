package mimetype

import (
	"strings"
)

type MimeType string

const (
	JSON = MimeType("application/json")
	BSON = MimeType("application/bson")
	YAML = MimeType("application/yaml")
	TEXT = MimeType("text/plain")
	// UNKNOWN is used when the incoming string is blank
	UNKNOWN = MimeType("")
)

var objectMimeTypes = []MimeType{JSON, BSON, YAML}

type headerFetcher interface {
	Get(string) string
}

func FromHeader(headers headerFetcher) MimeType {
	return FromString(headers.Get("Content-Type"))
}

func FromString(incoming string) MimeType {
	incoming = strings.ToLower(incoming)

	if incoming == "" {
		return UNKNOWN
	}
	if incoming == "text/plain" || incoming == "text" {
		return TEXT
	}

	for _, mimeType := range objectMimeTypes {
		mimeTypeLower := strings.ToLower(string(mimeType))
		mimeTypeLower = strings.Split(mimeTypeLower, "/")[1]
		if strings.HasSuffix(incoming, mimeTypeLower) {
			return mimeType
		}
	}

	return MimeType(incoming)
}
