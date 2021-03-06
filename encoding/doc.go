// Arbitrarily encode and decode of message body content.
/*
Spantool's goal is to make a single interface specification for any given content type,
so that content can be determined and decoded dynamically based on message headers or
mimetype sniffing, removing the need to explicitly call mimetype-specific methods when
decoding content.

Specific objectives

1. Clients can send arbitrary object serializations and request back whichever encoding
type they are most comfortable with.

2. Service developers do not have to explicitly add support for encoding types to a
given service or handler. Support for a mimetype should be able to be added once to
a shared library and gotten for free by an entire ecosystem.

3. Content encoding and decoding support should be independent of service pattern. IE,
adding support for understanding yaml should upgrade both REST server and http client
libraries on a rebuild.

4. Developers can easily extend all of their services to support a new content type
by creating their own encoding.

5. All default encoding shipped with spantools can be easily extendable to handle
custom types.
*/
package encoding
