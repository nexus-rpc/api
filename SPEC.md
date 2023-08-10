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

- [Service Name](#service-name)
- [Operation Name](#operation-name)
- [Operation ID](#operation-id)

> **Note:** More details regarding IDs will be provided upon finalizing the ID proposal.

### Service Name

Service names:

- MUST not be empty.
- MUST consist of only unreserved URL characters as delineated in [RFC3986 Section 2.3][rfc3986-section-2.3].

Valid characters include:

```
ALPHA / DIGIT / "-" / "." / "_" / "~"
```

### Operation Name

Operation names:

- MUST not be empty.
- MUST consist of only unreserved URL characters as delineated in [RFC3986 Section 2.3][rfc3986-section-2.3].

Valid characters include:

```
ALPHA / DIGIT / "-" / "." / "_" / "~"
```

### Operation ID

Operation IDs:

- MUST not be empty.
- MUST consist of only unreserved URL characters as delineated in [RFC3986 Section 2.3][rfc3986-section-2.3].

Valid characters include:

```
ALPHA / DIGIT / "-" / "." / "_" / "~"
```

## Schema Definitions

### Failure

The `Failure` object represents outcomes that are `failed` or `canceled`. The object MUST adhere to the following JSON
schema:

```yaml
type: object
properties:
  message:
    type: string
  details:
    type: object
    properties:
      metadata:
        type: object
        additionalProperties:
          type: string
        description: |
          String to string mapping, may contain information for decoding the data field.
      data:
        type: string
        description: |
          Arbitrary string-encoded data; use encodings like base64 for binary data transmission.
```

### OperationStarted

An `OperationStarted` object indicates the commencement of an asynchronous operation. This object MUST follow the given
JSON schema:

```yaml
type: object
properties:
  operationId:
    type: string
    description: |
      An identifier for referencing this operation. If provided in the start request, it is echoed back.
  callbackUrlSupported:
    type: boolean
    description: |
      If the request specified a callback_url and the handler supports callbacks for this operation, this flag will be
      set.
      It is up to the caller to decided how to get the outcome of this call in case the handler does not support
      callbacks.
```

### OperationInfo

The `OperationInfo` object, retrieved from the get operation info API, MUST adhere to the given schema:

```yaml
type: object
properties:
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

Start an arbirary length operation.
The response of the operation may be delivered synchronously (inline), or asynchronously, via a provided callback or the
[Get Operation Result](#get-operation-result) endpoint.

#### Paths

- `/api/v1/services/{service}/operations/{operation}`
- `/api/v1/services/{service}/operations/{operation}/{operation_id}`

#### Method

`POST`

#### Query Parameters

- `callback_url`: Optional. If the operation is asynchronous and the handler supports callbacks, it should invoke this URL once the operation's result is available.

#### Request Body

The body may contain arbitrary data. Headers should specify content type and encoding.

#### Response Codes

- `200 OK`: Operation completed successfully.

  **Headers**:

  - `Nexus-Operation-State: succeeded`

  **Body**: Arbitrary data conveying the operation's result. Headers should specify content type and encoding.

- `204 No Content`: Operation completed successfully with an empty result.

  **Headers**:

  - `Nexus-Operation-State: succeeded`

  **Body**: Empty.

- `482 Operation Failed`: Operation completed as `failed` or `canceled`.

  **Headers**:

  - `Content-Type: application/json`
  - `Nexus-Operation-State: failed | canceled`

  **Body**: A JSON serialized [`Failure`](#failure) object.

- `201 Created`: Operation was started an will completed asynchronously.

  **Headers**:

  - `Content-Type: application/json`

  **Body**: A JSON serialized [`OperationStarted`](#operationstarted) object.

### Cancel Operation

Request to cancel an operation.
The operation may later complete as canceled or any other outcome.
Handlers should ignore multiple cancelations of the same operation and return successfully if cancelation was already
requested.

#### Method

`POST`

#### Paths

- `/api/v1/services/{service}/operations/{operation}/{operation_id}/cancel`

#### Response Codes

- `202 Accepted`: Cancelation request accepted.

  **Body**: Unspecified.

- `404 Not Found`: Operation ID not recognized or references deleted.

  **Body**: Unspecified.

### Get Operation Result

Retrieve operation result.

#### Method

`GET`

#### Paths

- `/api/v1/services/{service}/operations/{operation}/{operation_id}/result`

#### Query Parameters

- `wait_deadline`: Optional. ISO 8601 date indicating the waiting period for a result.
  If not specified, and the operation is still running, the request should resolve immediately with the current
  operation status.

#### Response Codes

- `200 OK`: Operation completed successfully.

  **Headers**:

  - `Nexus-Operation-State: succeeded`

  **Body**: Arbitrary data conveying the operation's result. Headers should specify content type and encoding.

- `204 No Content`: Operation completed successfully with an empty result or is still running.

  **Headers**:

  - `Nexus-Operation-State: succeeded | running`

  **Body**: Empty.

- `482 Operation Failed`: Operation completed as `failed` or `canceled`.

  **Headers**:

  - `Content-Type: application/json`
  - `Nexus-Operation-State: failed | canceled`

  **Body**: A JSON serialized [`Failure`](#failure) object.

- `408 Request Timeout`: The specified `wait_deadline` expired prior to operation completion.

  The caller may re-issue this request to start a new long poll.

  **Body**: Unspecified.

- `404 Not Found`: Operation ID not recognized or references deleted.

  **Body**: Unspecified.

### Get Operation Info

Retrieve operation details.

#### Method

`GET`

#### Paths

- `/api/v1/services/{service}/operations/{operation}/{operation_id}`

#### Response Codes

- `200 OK`: Successfully retrieved info.

  **Headers**:

  - `Content-Type: application/json`
  - `Nexus-Operation-State: failed | canceled`

  **Body**:

  A JSON serialized [`OperationInfo`](#operationinfo) object.

- `404 Not Found`: Operation ID not recognized or references deleted.

## General Information on HTTP Response Codes

The Nexus protocol follows standard HTTP practices:

- `4xx` codes signify non-retryable framework-level errors.
- `5xx` codes signify retryable framework-level errors.

## Callback URLs

Any HTTP URL can be used to deliver operation completions.

Callers should ensure URLs contain sufficient information to correlate completions with initiators.

For invoking a callback URL:

- Include the `Nexus-Operation-State` header.
- If state is `succeeded`, deliver non-empty results in the body with corresponding `Content-*` headers.
- If state is `failed` or `canceled`, content type should be `application/json` and the body must have a serialized [`Failure`](#failure) object.
- Upon successful completion delivery, the handler should reply with a `200 OK` status and an empty body.

### Security

There's no enforced security for callback URLs.
When starting an operation, callers may embed a signed token into the URL, which can be verified upon delivery of completion.

[rfc3986-section-2.3]: https://datatracker.ietf.org/doc/html/rfc3986#section-2.3

### Q/A

1. What is the purpose of the `482 Operation Failed` response code? This is not a standard HTTP response code.

In internal discussions, we determined that there's value in having a specific status code for denoting operation
failures.
We chose a 4xx code because it is not meant to be retried and can be used by standard HTTP libraries and tools to
easier distinguish failure vs. success.
Non of the standard HTTP codes fit and we decided to define a custom status code.

2. What potential security concerns should be taken into consideration while implementing this protocol?

Security is not part of this specification, but as this is a thin layer on top of HTTP, standard practices should be
used to secure these APIs.
