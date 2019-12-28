// Named types for encoding / decoding extensions.
package spantypes

import (
	"encoding/hex"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"
)

// BinData is used to hold raw binary blob information for structs that need to support
// encoding to and from JSON / BSON. The json encoder will hexify this data for
// transport, while BSON will transform it to a BSON Binary primitive.
type BinData []byte

// Marshal to text value for json and others that implement this interface.
func (data BinData) MarshalText() ([]byte, error) {
	bytesValue := make([]byte, hex.EncodedLen(len(data)))
	encodedLen := hex.Encode(bytesValue, data)
	if hex.DecodedLen(encodedLen) != len(data) {
		return nil, xerrors.New("error encoding BinData to hex")
	}
	return bytesValue, nil
}

// UnMarshal from text value for json and others that implement this interface.
func (data *BinData) UnmarshalText(incomingData []byte) error {
	// Expand our data to have the length of the incoming bytes.
	*data = make([]byte, hex.DecodedLen(len(incomingData)))

	// Decode the hex value
	_, err := hex.Decode(*data, incomingData)
	if err != nil {
		return xerrors.Errorf("could not decode hex: %w", err)
	}

	return nil
}

// Marshal bson value.
func (data BinData) MarshalBSONValue() (bsontype.Type, []byte, error) {
	encoded := primitive.Binary{
		Subtype: 0x0,
		Data:    data,
	}
	return bson.MarshalValue(encoded)
}

// Unmarshal bson value.
func (data *BinData) UnmarshalBSONValue(
	valueType bsontype.Type, incomingData []byte,
) error {
	subType, rawData, _, ok := bsoncore.ReadBinary(incomingData)
	if !ok {
		return xerrors.New("unknown error decoding spantools.BinData")
	}
	if subType != 0x0 {
		return xerrors.New("spantools.BinData field is not bson subtype 0x0")
	}

	*data = rawData
	return nil
}
