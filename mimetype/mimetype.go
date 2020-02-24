// Enumeration-like type for content mimetypes.
package mimetype

import (
	"strings"
)

/*
MimeType is used to enumerate the default representation for content encoding types.
Non default MimeTypes can be used by wrapping a custom string:

	MimeType("text/csv")
*/
type MimeType string

const (
	JSON = MimeType("application/json")
	BSON = MimeType("application/bson")
	YAML = MimeType("application/yaml")
	TEXT = MimeType("text/plain")
	// UNKNOWN is used when the incoming string is blank
	UNKNOWN = MimeType("")
)

// List of default mimeTypes that are encoded to / from objects (as opposed to raw
// text).
var objectMimeTypes = []MimeType{JSON, BSON, YAML}

// Interface for object used to set headers such as http.Request.Header or
// http.Response.Header
type headerFetcher interface {
	Get(string) string
}

// Extract content type from a message / request header.
func FromHeader(headers headerFetcher) MimeType {
	return FromString(headers.Get("Content-Type"))
}

/*
Convert MimeType from a string. Ignores case. If the MimeType is a default type,
multiple formats are respected. For instance, all of the following will yield
"mimetype.JSON":

• "application/json"``

• "application/JSON"

• "application/x-json"

• "json"

• "x-json"
*/
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
