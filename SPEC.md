# Nexus RPC HTTP Specification

## Overview

The Nexus protocol, as specified below, is a synchronous RPC protocol. Arbitrary length operations are modelled on top
of a set of pre-defined synchronous RPCs.

A Nexus **caller** calls a **handler**. The handler may respond inline or return a reference for a future, asynchronous
operation. The caller can cancel an asynchronous operation, check for its outcome, or fetch its current state. The caller
can also specify a callback URL, which the handler uses to asynchronously deliver the result of an operation when it
is ready.

## Operation Addressability

An operation is addressed using three components:

- The containing service, a URL prefix (e.g. `http://api.mycompany.com/v1/myservice/`)
- [Operation Name](#operation-name)
- [Operation ID](#operation-id)

### Operation Name

Operation names:

- MUST not be empty.
- MUST consist of only unreserved URL characters as delineated in [RFC3986 Section 2.3][rfc3986-section-2.3].

Valid characters include:

```
ALPHA / DIGIT / "-" / "." / "_" / "~"
```

### Operation ID

A handler assigned identifier, returned from a [Start Operation](#start-operation) call.

Operation IDs:

- MUST not be empty.
- MUST consist of only unreserved URL characters as delineated in [RFC3986 Section 2.3][rfc3986-section-2.3].

Valid characters include:

```
ALPHA / DIGIT / "-" / "." / "_" / "~"
```

## Schema Definitions

### Failure

The `Failure` object represents protocol level failures returned in non successful HTTP responses as well as `failed` or
`canceled` operation results. The object MUST adhere to the following JSON schema:

```yaml
type: object
properties:
  message:
    type: string
    description: A simple text message.

  metadata:
    type: object
    additionalProperties:
      type: string
    description: |
      A key-value mapping for additional context. Useful for decoding the 'details' field, if needed. For example, to
      indicate base64 encoded data in 'details', set metadata["Content-Transfer-Encoding"] to 'base64'.

  details:
    type: any
    properties:
    description: |
      Additional structured data. If this is byte data, it should be base64 encoded.
```

### OperationInfo

The `OperationInfo` object MUST adhere to the given schema:

```yaml
type: object
properties:
  id:
    type: string
    description: |
      An identifier for referencing this operation.

  state:
    enum:
      - succeeded
      - failed
      - canceled
      - running
    description: |
      Describes the current state of an operation.
```

## Endpoint Descriptions

### Start Operation

Start an arbitrary length operation.
The response of the operation may be delivered synchronously (inline), or asynchronously, via a provided callback or the
[Get Operation Result](#get-operation-result) endpoint.

**Path**: `/{operation}`

**Method**: `POST`

#### Query Parameters

- `callback_url`: Optional. If the operation is asynchronous, the handler should invoke this URL once the operation's
  result is available.

#### Request Headers

- `Nexus-Request-Id`: Unique ID used to dedupe starts. Callers MUST not reuse request IDs to start operations with
  different inputs. Handlers MUST reject requests that map to the same operation with different request IDs.

#### Request Body

The body may contain arbitrary data. Headers should specify content type and encoding.

#### Response Codes

- `200 OK`: Operation completed successfully.

  **Headers**:

  - `Nexus-Operation-State: succeeded`

  **Body**: Arbitrary data conveying the operation's result. Headers should specify content type and encoding.

- `201 Created`: Operation was started and will complete asynchronously.

  **Headers**:

  - `Content-Type: application/json`

  **Body**: A JSON serialized [`OperationInfo`](#operationinfo) object.

- `482 Operation Failed`: Operation completed as `failed` or `canceled`.

  **Headers**:

  - `Content-Type: application/json`
  - `Nexus-Operation-State: failed | canceled`

  **Body**: A JSON serialized [`Failure`](#failure) object.

- `409 Conflict`: This operation was already started with a different request ID.

  **Headers**:

  - `Content-Type: application/json`

  **Body**: A JSON serialized [`Failure`](#failure) object.

### Cancel Operation

Request to cancel an operation.
The operation may later complete as canceled or any other outcome.
Handlers should ignore multiple cancelations of the same operation and return successfully if cancelation was already
requested.

**Path**: `/{operation}/{operation_id}/cancel`

**Method**: `POST`

#### Response Codes

- `202 Accepted`: Cancelation request accepted.

  **Body**: Empty.

- `404 Not Found`: Operation ID not recognized or references deleted.

  **Headers**:

  - `Content-Type: application/json`

  **Body**: A JSON serialized [`Failure`](#failure) object.

### Get Operation Result

Retrieve operation result.

**Path**: `/{operation}/{operation_id}/result`

**Method**: `GET`

#### Query Parameters

- `wait`: Optional. Duration indicating the waiting period for a result, defaulting to no wait.
  If by the end of the wait period the operation is still running, the request should resolve with a 204 status code
  (see below).

  Format of this parameter is number + unit, where unit can be `ms` for milliseconds, `s` for seconds, and `m` for
  minutes. Examples:

  - `100ms`
  - `1m`
  - `5s`

#### Response Codes

- `200 OK`: Operation completed successfully.

  **Headers**:

  - `Nexus-Operation-State: succeeded`

  **Body**: Arbitrary data conveying the operation's result. Headers should specify content type and encoding.

- `204 No Content`: Operation completed successfully with an empty result or is still running.

  When waiting for completion, the caller may re-issue this request to start a new long poll.

  **Headers**:

  - `Nexus-Operation-State: running`

  **Body**: Empty.

- `482 Operation Failed`: Operation completed as `failed` or `canceled`.

  **Headers**:

  - `Content-Type: application/json`
  - `Nexus-Operation-State: failed | canceled`

  **Body**: A JSON serialized [`Failure`](#failure) object.

- `404 Not Found`: Operation ID not recognized or references deleted.

  **Headers**:

  - `Content-Type: application/json`

  **Body**: A JSON serialized [`Failure`](#failure) object.

### Get Operation Info

Retrieve operation details.

**Path**: `/{operation}/{operation_id}`

**Method**: `GET`

#### Response Codes

- `200 OK`: Successfully retrieved info.

  **Headers**:

  - `Content-Type: application/json`

  **Body**:

  A JSON serialized [`OperationInfo`](#operationinfo) object.

- `404 Not Found`: Operation ID not recognized or references deleted.

  **Headers**:

  - `Content-Type: application/json`

  **Body**: A JSON serialized [`Failure`](#failure) object.

## General Information on HTTP Response Codes

The Nexus protocol follows standard HTTP practices, response codes not specified here should be interpreted according to
the HTTP specification.

## Callback URLs

Any HTTP URL can be used to deliver operation completions.

Callers should ensure URLs contain sufficient information to correlate completions with initiators.

For invoking a callback URL:

- Issue a POST request to the caller-provided URL.
- Include the `Nexus-Operation-State` header.
- If state is `succeeded`, deliver non-empty results in the body with corresponding `Content-*` headers.
- If state is `failed` or `canceled`, content type should be `application/json` and the body must have a serialized
  [`Failure`](#failure) object.
- Upon successful completion delivery, the handler should reply with a `200 OK` status and an empty body.

### Security

There's no enforced security for callback URLs at this time. However, some specific Nexus server implementations may
deliver additional details as headers or have other security requirements of the callback endpoint.
When starting an operation, callers may embed a signed token into the URL, which can be verified upon delivery of
completion.

[rfc3986-section-2.3]: https://datatracker.ietf.org/doc/html/rfc3986#section-2.3

## Q/A

1. What is the purpose of the `482 Operation Failed` response code? This is not a standard HTTP response code.

In internal discussions, we determined that there's value in having a specific status code for denoting operation
failures.
We chose a 4xx code because it is not meant to be retried and can be used by standard HTTP libraries and tools to
easier distinguish failure vs. success.
Non of the standard HTTP codes fit and we decided to define a custom status code.

2. What potential security concerns should be taken into consideration while implementing this protocol?

Security is not part of this specification, but as this is a thin layer on top of HTTP, standard practices should be
used to secure these APIs. For securing callback URLs, see [Callback URLs > Security](#security).
