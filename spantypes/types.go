package spantypes

// BinData is used to hold raw binary blob information for structs that need to support
// encoding to and from JSON / BSON. The json encoder will hexify this data for
// transport, while BSON will transform it to a BSON Binary primitive.
type BinData []byte
