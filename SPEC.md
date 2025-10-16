# Nexus RPC HTTP Specification

## Overview

The Nexus protocol, as specified below, is a synchronous RPC protocol for system-to-system integration. Arbitrary
duration operations are modelled on top of a set of pre-defined synchronous RPCs.

A Nexus **caller** calls a **handler**. The handler may respond inline (synchronous response) or return a token
referencing the ongoing operation (asynchronous response), which the caller may use to cancel the operation. In lieu of
a higher level service contract, the caller cannot determine whether an operation is going to resolve synchronously or
asynchronously, and should specify a callback URL, which the handler uses to deliver the result of an asynchronous
operation when it is ready.

## Operation Addressability

An operation is addressed using the following components:

- The containing endpoint, a URL prefix (e.g. `http://api.mycompany.com/services/`)
- Service Name - A grouping of operations (e.g. `payments.v1`)
- Operation Name - A unique name for the given (e.g. `charge`)
- Operation Token - A unique token assigned by the handler as a response to a [StartOperation](#start-operation)
  request.

The service name and operation name MUST not be empty and may contain any arbitrary character sequence as long as
they're encoded into the URL.

An operation token MUST not be empty and contain only characters that are valid HTTP header values. When passing a token
as part of a URL path make sure any special characters are encoded.

## Schema Definitions

All schemas in this specification follow the [JSON Schema](https://json-schema.org/specification) specification.

### Failure

The `Failure` object represents protocol-level failures returned in non-successful HTTP responses, as well as `failed`
or `canceled` operation results. The object MUST adhere to the following schema:

```yaml
type: object
properties:
  message:
    type: string
    description: A simple text message.

  stackTrace:
    type: string
    description: An optional stack trace that may be emitted in languages that support it.

  metadata:
    type: object
    additionalProperties:
      type: string
    description: |
      A key-value mapping for additional context. Useful for decoding the 'details' field, if needed.

  details:
    type: any
    properties:
    description: |
      Additional JSON-serializable structured data.

  cause:
    # An optional nested failure structure.
    $ref: '#'
```

### OperationInfo

The `OperationInfo` object MUST adhere to the following schema:

```yaml
type: object
properties:
  token:
    type: string
    description: |
      A token for referencing the operation.

  state:
    enum:
      - succeeded
      - failed
      - canceled
      - running
    description: |
      Describes the current state of the operation.
```

## Endpoint Descriptions

### Start Operation

Start an arbitrary duration operation. The response of the operation may be delivered synchronously (inline), or
asynchronously, via a provided callback or the [Fetch Operation Result](#fetch-operation-result) endpoint.

**Path**: `/{service}/{operation}`

**Method**: `POST`

#### Query Parameters

- `callback`: Optional. If the operation is asynchronous, the handler should invoke this URL once the operation's result
  is available.

#### Request Headers

A client may attach arbitrary headers to the request.

Headers that start with the `Nexus-Callback-` prefix are expected to be attached to the callback request when invoked by
the handler. The callback request must strip away the `Nexus-Callback-` prefix. E.g if a Start Operation request
includes a `Nexus-Callback-Token: some-token` header, the callback request would include a `Token: some-token` header.

If a callback query parameter is provided, the `Nexus-Callback-Token` header is **REQUIRED**. It MUST contain a caller-generated token that uniquely identifies the
originating operation. Handlers MUST include this header’s value as a `Token` header in all callback requests to the
caller-provided `callback` URL. This header allows correlation between callback requests and their originating
operations.

The `Operation-Timeout` header field can be added to inform the handler how long the caller is willing to wait for an
operation to complete. This is distinct from the more general `Request-Timeout` header which is used to indicate the
timeout for a single HTTP request. Format of this header value is number + unit, where unit can be `ms` for
milliseconds, `s` for seconds, and `m` for minutes.

The `Nexus-Link` header field can be added to associate resources with the start request. A handler may attach these
links as metadata to underlying resources to provide end-to-end observability. See the [`Nexus-Link`](#nexus-link)
section for more information.

#### Request Body

The body may contain arbitrary data. Headers should specify content type and encoding.

#### Response Codes

- `200 OK`: Operation completed successfully. It may return `Nexus-Link` headers linking to resources associated with
  this operation.

  **Headers**:

  - `Nexus-Operation-State: succeeded`

  **Body**: Arbitrary data conveying the operation's result. Headers should specify content type and encoding.

- `201 Created`: Operation was started and will complete asynchronously. It may return `Nexus-Link` headers to associate
  resources with this operation.

  **Headers**:

  - `Content-Type: application/json`

  **Body**: A JSON-serialized [`OperationInfo`](#operationinfo) object.

- `424 Failed Dependency`: Operation completed as `failed` or `canceled`.

  **Headers**:

  - `Content-Type: application/json`
  - `Nexus-Operation-State: failed | canceled` (DEPRECATED)

  **Body**: A JSON-serialized [`Failure`](#failure) object representing an [`OperationError`](#operation-error).

### Cancel Operation

Request to cancel an operation. The operation may later complete as canceled or any other outcome. Handlers should
ignore multiple cancellations of the same operation and return successfully if cancellation was already requested.

**Path**: `/{service}/{operation}/cancel`

**Method**: `POST`

#### Request Headers

The operation token received as a response to the Start Operation method must be delivered either via the
`Nexus-Operation-Token` header field or the `token` query param.

#### Query Parameters

- `token`: The operation token received as a response to the Start Operation method. Must be delivered either via the
  `token` query param or the `Nexus-Operation-Token` header field.

#### Response Codes

- `202 Accepted`: Cancelation request accepted.

  **Body**: Empty.

## Predefined Failure Types

### `OperationError`

An Operation Error represents a failed or canceled operation outcome. It may be returned responses to `StartOperation`
requests, and in the body of async completion requests.

Operation Error [`Failure`](#failure) representation is as follows:

```json
{
  "metadata": {
    "type": "nexus.OperationError",
  },
  "message": "<Optional error message>",
  "stackTrace": "<Optional stack trace>",
  "cause": { /* <Optional cause> */ },
  "details": {
    "state": "canceled | failed",
    // Arbitrary details may be added here as needed.
  },
}
```

### `HandlerError`

A HandlerError represents errors while handling a request. They include an error type as defined
[below](#predefined-handler-errors). Each error type has predefined, overridable, retry semantics. Handler Errors can be
returned responses to any of the methods defined above as well as completion callbacks.

Handler Error [`Failure`](#failure) representation is as follows:

```json
{
  "metadata": {
    "type": "nexus.HandlerError",
  },
  "message": "<Optional error message>",
  "stackTrace": "<Optional stack trace>",
  "cause": { /* <Optional cause> */ },
  "details": {
    "type": "<predefined error type (e.g. INTERNAL)",
    // "retryableOverride": Optional boolean.
    // Arbitrary details may be added here as needed.
  },
}
```

### Predefined Handler Errors

For compatibility of this HTTP spec with future transports, when a handler fails a request, it **should** use one of the
following predefined error codes.

| Name                 | Status Code | Description                                                                                                                                                                                                                                                  |
| -------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `BAD_REQUEST`        | 400         | The handler cannot or will not process the request due to an apparent client error. Clients should not retry this request unless advised otherwise.                                                                                                          |
| `UNAUTHENTICATED`    | 401         | The client did not supply valid authentication credentials for this request. Clients should not retry this request unless advised otherwise.                                                                                                                 |
| `UNAUTHORIZED`       | 403         | The caller does not have permission to execute the specified operation. Clients should not retry this request unless advised otherwise.                                                                                                                      |
| `NOT_FOUND`          | 404         | The requested resource could not be found but may be available in the future. Subsequent requests by the client are permissible but not advised.                                                                                                             |
| `REQUEST_TIMEOUT`    | 408         | Returned by the server to when it has given up handling a request. This may occur by enforcing a client provided `Request-Timeout` or for any arbitrary reason such as enforcing some configurable limit. Subsequent requests by the client are permissible. |
| `CONFLICT`           | 409         | The request could not be made due to a conflict. This may happen when trying to create an operation that has already been started. Clients should not retry this request unless advised otherwise.                                                           |
| `RESOURCE_EXHAUSTED` | 429         | Some resource has been exhausted, perhaps a per-user quota, or perhaps the entire file system is out of space. Subsequent requests by the client are permissible.                                                                                            |
| `INTERNAL`           | 500         | An internal error occurred. Subsequent requests by the client are permissible.                                                                                                                                                                               |
| `NOT_IMPLEMENTED`    | 501         | The handler either does not recognize the request method, or it lacks the ability to fulfill the request. Clients should not retry this request unless advised otherwise.                                                                                    |
| `UNAVAILABLE`        | 503         | The service is currently unavailable. Subsequent requests by the client are permissible.                                                                                                                                                                     |
| `UPSTREAM_TIMEOUT`   | 520         | Used by gateways to report that a request to an upstream handler has timed out. Subsequent requests by the client are permissible.                                                                                                                           |

Client implementations should try to rehydrate a `HandlerError` from the serialized `Failure` object in the response
body whenever a request fails with one of the status codes listed below. If the handler error type in the `Failure`
object details doesn't match the response status code, the `Failure` object takes precedence.

If the serialized `Failure` does not represent a `HandlerError`, clients should construct a wrapper `HandlerError`,
setting the response `Failure` as the `cause` (if available) translating the response status code to the `HandlerError`
type and status text to the message.

## General Purpose Headers

### `Nexus-Link`

The `Nexus-Link` header field provides a means for serializing one or more links in HTTP headers. This header is encoded
the same way as the HTTP header `Link` described [here](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Link).

Handlers and callers can specify links in Nexus requests and responses to associate an operation with arbitrary
resources.

Links must contain a `type` parameter that expresses how they should be parsed.

**Example**: `Nexus-Link: <myscheme://somepath?k=v>; type="com.example.MyResource"`

### `Nexus-Request-Retryable` (DEPRECATED)

Handlers may specify the `Nexus-Request-Retryable` header with a value of `true` or `false` to explicitly instruct a
caller whether or not to retry a request. Unless specified, retry behavior is determined by the
[predefined handler error type](#predefined-handler-errors).

### `Request-Timeout`

Callers may specify the `Request-Timeout` header on all APIs to inform the handler how long they're willing to wait for
a response.

Format of this header value is number + unit, where unit can be `ms` for milliseconds, `s` for seconds, and `m` for
minutes.

## Callback URLs

Any HTTP URL can be used to deliver operation completions.

Callers should ensure URLs contain sufficient information to correlate completions with initiators.

For invoking a callback URL:

- Issue a POST request to the caller-provided URL.
- Include any callback headers supplied in the originating StartOperation request, stripping away the `Nexus-Callback-`
  prefix.

- The callback request **MUST include** a `Token` header derived from the required `Nexus-Callback-Token` header in the
  originating StartOperation request. This header uniquely associates the callback with its originating operation and is
  mandatory for all callback deliveries.

- Include the following headers for resources associated with this operation to support completing asynchronous
  operations before the response to StartOperation is received:
  - `Nexus-Operation-Token`
  - `Nexus-Operation-Start-Time`
  - any `Nexus-Link` headers
- The `Nexus-Operation-Start-Time` header should be in a valid HTTP format described
  [here](https://www.rfc-editor.org/rfc/rfc5322.html#section-3.3). If omitted, the time the completion is received will
  be used as operation start time.
- Include a `Nexus-Operation-Close-Time` header, indicating the time when the operation completed (either successfully
  or unsuccessfully). The header's value must be a
  [valid RFC 3339 format timestamp](https://datatracker.ietf.org/doc/html/rfc3339#section-5), with a resolution of
  milliseconds or finer.
  - RFC 3339 timestamps and ISO 8601 timestamps are usually compatible, but some edge cases may apply
    [[1](https://protobuf.dev/programming-guides/json/#rfc3339), [2](https://ijmacd.github.io/rfc3339-iso8601/)].
- Include the `Nexus-Operation-State` header.
- If state is `succeeded`, deliver non-empty results in the body with corresponding `Content-*` headers.
- If state is `failed` or `canceled`, content type should be `application/json` and the body must have a serialized
  [`Failure`](#failure) object.
- Upon successful completion delivery, the handler should reply with a `200 OK` status and an empty body.

### Security

There's no enforced security for callback URLs at this time. However, some specific Nexus server implementations may
deliver additional details as headers or have other security requirements of the callback endpoint. When starting an
operation, callers may embed a signed token into the URL, which can be verified upon delivery of completion.

[rfc3986-section-2.3]: https://datatracker.ietf.org/doc/html/rfc3986#section-2.3

## Content Types

Nexus is not opinionated about request content types, although typically, inputs and outputs are transmitted as JSON
with content type `application/json`. Other common types include nulls, where the content type is left empty, and byte
buffers, with content type `application/octet-stream`.

### Protocol Buffers

[Protobuf](https://protobuf.dev/) messages support two serialization formats, binary, and JSON.

The standard way to transmit binary serialized protos over Nexus is to attach the following `content-type` header:

```
application/x-protobuf; message-type=com.example.MyMessage
```

The standard way to transmit JSON serialized protos over Nexus is to attach the following `content-type` header:

```
application/json; format=protobuf; message-type=com.example.MyMessage
```

Note that in both cases, the message-type is the fully qualified proto message name. The message-type param is used to
look up the message in the proto registry in languages that do not have runtime type information. Languages that do have
support for runtime types, should validate that the message type in the header matches the value to deserialize into.

## Q/A

1. What potential security concerns should be taken into consideration while implementing this protocol?

Security is not part of this specification, but as this is a thin layer on top of HTTP, standard practices should be
used to secure these APIs. For securing callback URLs, see [Callback URLs > Security](#security).
