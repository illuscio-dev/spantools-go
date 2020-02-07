Spantools-Go
============

Spantools is a shared library for enabling spanreed-style communication between
microservices.

Encoding Goals
==============

spantools/encoding's goal is to make a single interface specification for any given
content type, and enable the following features:

#. Services can determine request and response content types dynamically based on
   message headers or mimetype sniffing.

#. Remove need to call encoding-specific methods for decode.

#. Clients can send arbitrary object serializations and request back whichever
   encoding type they are most comfortable with.

#. Service developers do not have to explicitly add support for encoding types to a
   given service or handler. Support for a mimetype should be able to be added once
   to a shared library and gotten for free by an entire ecosystem.

#. Content encoding and decoding support should be independent of service pattern.
   IE, adding support for understanding yaml should upgrade both REST server and
   http client libraries on a rebuild.

#. Developers can easily extend all of their services to support a new content type
   by creating their own encoding.

#. All default encoding shipped with spantools can be easily extended to handle
   custom types and types from third-party packaged used by the service.

#. When possible, developers are able to define handlers for third party types for each
   encoding type. This allows the use of convenient third party data types that do not
   define a marshaller or unmarshaller function for a given encoding without having to
   wrap them in a new type.

Encoding Quickstart
===================

Spantools offers a consistent interface for encoding / decoding arbitrary content
encoding types.

Declare a ContentEngine
-----------------------

.. code-block:: go

    package main

    import (
        "bytes"
        "fmt"
        "spantools/encoding"
        "spantools/mimetype"
    )

In the ``main()`` function, declare your engine:

.. code-block:: go

    engine, err := encoding.NewContentEngine(true)
    if err != nil{
        panic("error creating content engine")
    }


Decode a Payload
----------------

.. code-block:: go

    // Make our json content to decode
    content := []byte(`{"name": "Harry Potter", "house": "Potter"}`)
    contentBuffer := bytes.NewBuffer(content)

    // Here's the type we are going to decode to
    type Wizard struct {
        Name string `json:"name" bson:"name"`
        House string `json:"house" bson:"name"`
    }

    // Declare a pointer to the wizard we want to unmarshall our json to
    student := new(Wizard)

    // Ask the engine to decode the content
    if err := engine.Decode(mimetype.JSON, student, contentBuffer) ; err != nil {
        panic(err)
    }

    // Display the result
    fmt.Printf("%+v\n", student)

Output: ::

    &{Name:Harry Potter House:Potter}

Decoding other types can be accomplished by changing ``mimetype.JSON`` to a different
value.


Decode a Payload of an Unknown Type
-----------------------------------

To decode content of an unknown mimetype, use:

.. code-block:: go

    if err := engine.Decode(mimetype.UNKNOWN, student, contentBuffer) ; err != nil {
        panic(err)
    }

The engine will attempt to decode the

Encode a Payload
----------------

Let's encode a struct to JSON:

.. code-block:: go

	wizard := &Wizard{
		Name:  "Draco Malfoy",
		House: "Slytherin",
	}

	contentBuffer := new(bytes.Buffer)
	if err := engine.Encode(mimetype.JSON, wizard, contentBuffer) ; err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", contentBuffer.String())

Encoding to BSON is as easy as changing the format:

.. code-block:: go

	if err := engine.Encode(mimetype.BSON, wizard, contentBuffer) ; err != nil {
        panic(err)
	}

BSON is offered as default format for faster fetching from MongoDB when a record just
need to be passed along.

Register a New Encoder
----------------------

Let's define an encoder/decoder for text/csv. First, create a type to implement the
encoder.

.. code-block:: go

    type YamlEncoder struct{}

    // This method implements the encoding.Decoder type.
    func (*YamlEncoder) Decode(
        engine encoding.ContentEngine, reader io.Reader, contentReceiver interface{},
    ) error {
        contentBuffer := new(bytes.Buffer)
        _, err := contentBuffer.ReadFrom(reader)
        if err != nil {
            return xerrors.Errorf("error reading content: %w", err)
        }

        // This encoder / decoder is going to be a wrapper around the go-yaml library.
        err = yaml.Unmarshal(contentBuffer.Bytes(), contentReceiver)
        if err != nil {
            return xerrors.Errorf("error unmarshalling yaml: %w", err)
        }
        return nil
    }

    // This method implements the encoding.Encoder type.
    func (*YamlEncoder) Encode(
        engine encoding.ContentEngine, writer io.Writer, content interface{},
    ) error {
        contentBytes, err := yaml.Marshal(content)
        if err != nil {
            return xerrors.Errorf("error marshalling yaml: %w", err)
        }

        _, err = writer.Write(contentBytes)
        if err != nil {
            return xerrors.Errorf("error writing yaml: %w", err)
        }
        return nil
    }

Define a data model to unmarshal from / marshal to:

.. code-block:: go

    type Wizard struct {
        Name string `yaml:"name"`
        House string `yaml:"house"`
    }

In our ``main()`` function, create a new engine and register the encoder to it:

.. code-block:: go

    engine, err := encoding.NewContentEngine(false)
    if err != nil {
        panic(err)
    }

    // Make the encoder / decoder instance
    yamlEncoder := new(YamlEncoder)

    // Register for encoding:
    engine.SetEncoder("application/yaml", yamlEncoder)

    // Register for decoding:
    engine.SetDecoder("application/yaml", yamlEncoder)

.. note::

    In the above example, we use ``"application/yaml"`` as the mimetype. You can also
    use ``mimetype.YAML`` which is an alias for ``"application/yaml"``.

Now lets test it:

.. code-block:: go

    yamlString := []byte("name: Hermione Granger\nhouse: Gryffindor")
    contentBuffer := bytes.NewBuffer(yamlString)

    // Lets decode it into a wizard
    wizard := new(Wizard)
    err = engine.Decode("application/yaml", wizard, contentBuffer)
    if err != nil {
        panic(err)
    }

    // Print the decoded object
    fmt.Printf("\nDECODED:\n%+v\n", wizard)

    // Encode the object into a YAML binary
    encodeBuffer := new(bytes.Buffer)
    err = engine.Encode("application/yaml", wizard, encodeBuffer)

    fmt.Printf("\nENCODED:\n%+v\n", encodeBuffer.String())

Output: ::

    DECODED:
    &{Name:Hermione Granger House:Gryffindor}

    ENCODED:
    name: Hermione Granger
    house: Gryffindor


Extend SpanEngine
-----------------

It's possible to extend the SpanEngine type.

Lets say we want an ContentEngine with an ``AppName`` field so that we can access it
in a custom encoder:

.. code-block:: go

    type CustomEngine struct {
        *encoding.SpanEngine
        AppName string
    }

Now lets define a text Encoder that uses the engine while dumping the content:

.. code-block:: go

    type CustomTextEncoder struct {}

    func (encoder CustomTextEncoder) Encode(
        engine encoding.ContentEngine, writer io.Writer, content interface{},
    ) error {
        // Make a type assert to convert the engine interface passed in to the encoder
        // to our engine type.
        ourEngine := engine.(*CustomEngine)

        // This Encoder is only going to accept strings, so we're going to assert the
        // type here.
        contentString := content.(string)
        contentString = ourEngine.AppName + " says: '" + contentString + "'."

        _, err := writer.Write([]byte(contentString))
        if err != nil {
            return xerrors.Errorf("error writing text to payload: %w", err)
        }
        return nil
    }

In our ``main()`` function we can make our new engine by embedding the default one:

.. code-block:: go

    engine, err := encoding.NewContentEngine(false)
    if err != nil {
        panic(err)
    }

    ourEngine := &CustomEngine{
        SpanEngine: engine,
        AppName: "MyAwesomeApp",
    }

Now we need to signal to the underlying spanEngine to pass in our custom engine to the
Encoder.Encode() and Encoder.Decode() app:

.. code-block:: go

    ourEngine.SetPassedEngine(ourEngine)

Register our encoder. It will replace the default text/plain encoder:

.. code-block:: go

    ourEngine.SetEncoder(mimetype.TEXT, &CustomTextEncoder{})

Now we can use the engine as normal:

.. code-block:: go

    buffer := new(bytes.Buffer)
    err = ourEngine.Encode(mimetype.TEXT, "some message", buffer)
    if err != nil {
        panic(err)
    }

    fmt.Println("ENCODED:", buffer.String())

Output: ::

    ENCODED: MyAwesomeApp says: 'some message'.


Register a JSON Extension
-------------------------

SpanEngine uses `go-codec`_ for json encoding and decoding which allows the registration of
`extensions <http://ugorji.net/blog/go-codec-primer#using-extensions>`_ for custom
named types.

Lets say we have a type defined in a third party package that looks like this:

.. code-block:: go

    type Fraction struct {
        Nominator int
        Denominator int
    }

We want to have access to it's methods, but it does not define any of the interfaces
that `go-codec`_ cues off of (json.Marshaler, json.Unmarshaler, json.TextMarshaler,
etc...). We can define an extension for it using
`go-codec's extensions <http://ugorji.net/blog/go-codec-primer#using-extensions>`_:

.. code-block:: go

    type FractionExtension struct {}

    // Encodes value to string with format 'nominator/denominator'.
    func (ext *FractionExtension) ConvertExt(value interface{}) interface{} {
        valueFraction := value.(*Fraction)

        valueString := strconv.Itoa(valueFraction.Nominator) +
            "/" + strconv.Itoa(valueFraction.Nominator)

        return valueString
    }

    // Decodes value from string with format 'nominator/denominator'.
    func (ext *FractionExtension) UpdateExt(dest interface{}, value interface{}) {
        destVal := dest.(*Fraction)
        fracString := value.(string)

        split := strings.Split(fracString, "/")
        if len(split) != 2 {
            panic(xerrors.New("could not convert '" + fracString + "' to string"))
        }

        nominator, err := strconv.Atoi(split[0])
        if err !=  nil {
            panic(
                xerrors.Errorf(
                    "error converting nominator of '" +
                        fracString + "' to frac: %w", err,
                ),
            )
        }

        denominator, err := strconv.Atoi(split[0])
        if err !=  nil {
            panic(
                xerrors.Errorf(
                    "error converting denominator of'" +
                        fracString + "' to frac: %w", err,
                ),
            )
        }

        *destVal = Fraction{
            Nominator:   nominator,
            Denominator: denominator,
        }
    }

Now lets create a list of extensions to register with the engine using the
encoding.JsonExtensionOpts type:

.. code-block:: go

    var jsonExtensions = []*encoding.JsonExtensionOpts{
        {
            // The value type this extension should act on
            ValueType:    reflect.TypeOf(Fraction{}),

            // The extension itself.
            ExtInterface: &FractionExtension{},
        },
    }

Now in our ``main()`` function we can create an engine and register our list of
extensions:

.. code-block:: go

    engine, err := encoding.NewContentEngine(false)
    if err != nil {
        panic(err)
    }

    err = engine.AddJsonExtensions(jsonExtensions)
    if err != nil {
        panic(err)
    }

Now lets encode an object with a fraction value:

.. code-block:: go

    type HasRational struct {
        Fraction Fraction
    }

    ourObject := &HasRational{Fraction: Fraction{1, 64}}

    buffer := new(bytes.Buffer)
    err = engine.Encode(mimetype.JSON, ourObject, buffer)
    if err != nil {
        panic(err)
    }

    fmt.Println("DUMPED:", buffer.String())

Output: ::

    DUMPED: {"Fraction":"1/1"}

And now lets decode it again:

.. code-block:: go

    decoded := new(HasRational)
    err = engine.Decode(mimetype.JSON, decoded, buffer)
    if err != nil {
        panic(err)
    }

    fmt.Printf("LOADED: %+v\n", decoded)

Output: ::

    LOADED: &{Fraction:{Nominator:1 Denominator:1}}

.. note::

    **Why go-codec extensions?**

    Spantools opts to use the go-codec/JsonHandler over the standard encoding/Json
    because it allows you to define extensions for arbitrary third-party types.

    We can make codecs to handle bson's primitives.Binary for instance rather than
    having to wrap it in a new type and define a marshal/unmarshal function on THAT,
    which in turn reduces the mental overhead of using some fields where our real
    type is embedded.

.. note::

    **Default Json Extensions**

    SpanEngine ships with the following types handled:

    • UUIDs from `go.uuid`_

    • Binary blob data represented as ``spantypes.BinData`` are represented as a hex
      string.

    • `Bson primitive.Binary`_ data will be encoded as a string for 0x3 subtype (UUID)
      and a hex string for 0x0 subtype (arbitrary binary data). Other subtypes are not
      currently supported and will panic.

    • `bson.Raw`_ is unmarshaled to a ``map[string]interface{}`` and THEN encoded to a
      json object. Included to enable the direct return of a Bson document from a mongo
      database.

.. note::

    Named types can implement json.Marshaler, json.TextMarshaler, or codec.Selfer for
    marshaling and their unmarshaling counterparts to handle encoding and decoding.
    ``spantypes.BinData`` is handled in this manner.

.. warning::

    **Bson Types: Encode-Only**

    The `bson.Raw`_ and `primitive.Binary`_ type extensions supply ENCODING-ONLY
    methods, and will panic if a stuct uses these types as a target. They are supplied
    to enable the direct conversion of bson documents to json, not to enable their use
    in business-logic structs.

    It is recommended that the following types be used as intermediaries in stucts:

    • Bson UUIDs (`primitive.Binary`_ subtype 0x3) should be represented with `go.uuid`_
      UUID objects. SpanEngine comes with codecs to seamlessly handle the conversion
      of this type in and out of BSON.

    • Bson Binary blob (primitive.Binary subtype 0x3) should be represented as
      ``spantypes.BinData``. The bson encoder has a codec to handle the encoding /
      decoding of this type in and out of Bson.


Register a BSON Codec
---------------------

The official bson driver has a system very similar to codec's extensions, which through
sheer happenstance is called a
`ValueCodec interface <https://godoc.org/go.mongodb.org/mongo-driver/bson/bsoncodec#ValueCodec>`_.

Lets take an example.

Say we have a type defined in a third-party package we want to handle:

.. code-block:: go

    type Fraction struct {
        Nominator int
        Denominator int
    }

We can define a codec for it like so:

.. code-block:: go

    type FractionCodec struct {}

    // Encodes value
    func (codec *FractionCodec) EncodeValue(
        encodeCTX bsoncodec.EncodeContext,
        valueWriter bsonrw.ValueWriter,
        value reflect.Value,
    ) error {
        var fractionValue *Fraction

        switch incomingType := value.Interface().(type) {
        case *Fraction:
            fractionValue = incomingType
        case Fraction:
            fractionValue = &incomingType
        default:
            return xerrors.New("Error encoding fraction.")
        }

        valueString := strconv.Itoa(fractionValue.Nominator) +
            "/" + strconv.Itoa(fractionValue.Nominator)

        return valueWriter.WriteString(valueString)
    }

    // Decodes value
    func (codec *FractionCodec) DecodeValue(
        decodeCTX bsoncodec.DecodeContext,
        valueReader bsonrw.ValueReader,
        value reflect.Value,
    ) error {
        fracString, err := valueReader.ReadString()
        if err != nil {
            return err
        }

        split := strings.Split(fracString, "/")
        if len(split) != 2 {
            return xerrors.New("could not convert '" + fracString + "' to string")
        }

        nominator, err := strconv.Atoi(split[0])
        if err !=  nil {
            return xerrors.Errorf(
                "error converting nominator of '" +
                    fracString + "' to frac: %w", err,
            )
        }

        denominator, err := strconv.Atoi(split[0])
        if err !=  nil {
            return xerrors.Errorf(
                "error converting denominator of'" +
                    fracString + "' to frac: %w", err,
            )
        }

        fraction := &Fraction{
            Nominator:   nominator,
            Denominator: denominator,
        }

        if value.Kind() == reflect.Ptr {
            value.Set(reflect.ValueOf(fraction))
        } else {
            value.Set(reflect.ValueOf(*fraction))
        }

        return nil
    }

Now lets create a list of codecs to register with the engine using the
encoding.BsonCodecOpts type:

.. code-block:: go

    var bsonCodecs = []*encoding.BsonCodecOpts{
        {
            ValueType:    reflect.TypeOf(Fraction{}),
            Codec: 		  &FractionCodec{},
        },
    }

In our ``main()`` function, create the engine and register the codecs:

.. code-block:: go

    engine, err := encoding.NewContentEngine(false)
    if err != nil {
        panic(err)
    }

    err = engine.AddBsonCodecs(bsonCodecs)
    if err != nil {
        panic(err)
    }

Here is the form type that has our fraction that we want to send / receive as bson:

.. code-block:: go

    type HasRational struct {
        Fraction Fraction
    }

Let's encode one:

.. code-block:: go

    ourObject := &HasRational{Fraction: Fraction{1, 64}}

    buffer := new(bytes.Buffer)
    err = engine.Encode(mimetype.BSON, ourObject, buffer)
    if err != nil {
        panic(err)
    }

    fmt.Println("DUMPED:", buffer.String())

Output: ::

    DUMPED: fraction1/1

Decoding it again:

.. code-block:: go

    decoded := new(HasRational)
    err = engine.Decode(mimetype.BSON, decoded, buffer)
    if err != nil {
        panic(err)
    }

    fmt.Printf("LOADED: %+v\n", decoded)

Output: ::

    LOADED: &{Fraction:{Nominator:1 Denominator:1}}

.. note::

    **Default Bson ValueCodecs**

    SpanEngine ships with the following types handled:

    • UUIDs from `go.uuid`_ are converted to/from `primitive.Binary`_ subtype 0x3.

    • Binary blob data represented as ``spantypes.BinData`` are converted to/from
      `primitive.Binary`_ subtype 0x0.

.. note::

    Named types can implement bsoncodec.ValueMarshaler and bsoncodec.ValueUnmarshaler
    to handle encoding and decoding. ``spantypes.BinData`` is handled in this manner.

.. note::

    **Handling Multiple Bson Docs**

    Bson does not define a top-level list object for sending multiple documents over the
    wire. When marshalling to / unmarshalling from a list of objects, SpanEngine uses
    the "\u241E" character to denote breaks between documents.

    The ``string`` version of this delimiter is held in the
    ``encoding.BsonListSepString`` const, and the ``[]byte`` version is
    ``encoding.BsonListBytes``

.. note::

    **Accessing the bson codec registry**

    If you wish to use the bson registry yourself for encoding / decoding to your
    database, it is made accessible through ``SpanEngine.BsonRegistry()``.

.. warning::

    Do not add codecs directly to the registry returned by
    ``SpanEngine.BsonRegistry()``. ``SpanEngine.AddBsonCodecs(bsonCodecs)`` also
    updates the JsonExtension for encoding the `bson.Raw`_ type to include all codecs
    being added. Side-stepping it may cause panics when dumping a database document
    directly to json.

API Errors
==========

APIError and exceptions that inherit from it are designed to be raised by spanserver
during the processing of a request, then transmitted back for handling.

API exceptions are found in this toolbox for libraries which wish to consume these
errors without installing spanserver and all of its dependencies.

These errors can be found in the ``spantools.errors_api`` package.

Paging Models
=============

Paging models are designed to help spanserver / spanclient communicate about paging
information, and can be found in the ``spantools.models``


API documentation
=================

API documentation is created using godoc and can be
`found here <_static/godoc-root.html>`_.

.. _API documentation: _static/godoc-root.html
.. _go-codec: http://ugorji.net/blog/go-codec-primer
.. _Bson primitive.Binary: https://godoc.org/go.mongodb.org/mongo-driver/bson/primitive#Binary
.. _primitive.Binary: https://godoc.org/go.mongodb.org/mongo-driver/bson/primitive#Binary
.. _bson.Raw: https://godoc.org/go.mongodb.org/mongo-driver/bson#Raw
.. _go.uuid: github.com/satori/go.uuid
