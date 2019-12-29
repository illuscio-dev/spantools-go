/*
Spanreed error model definition and default span errors.

The Spanreed family strives to have a consistent set of errors (and error communication)
conventions shared between all services and clients.

This module defines two main objects for handing errors:

• SpanErrorType defines an error type.

• SpanError is an instance of an error which contains a SpanErrorType.

Default SpanErrorType Variables

Several pointers to SpanErrorType definitions are included in this package.
*/
package spanerrors
