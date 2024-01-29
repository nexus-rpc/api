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

Both the name and ID MUST not be empty and may contain any arbitrary character sequence as long as they're encoded into
the URL.

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
      A key-value mapping for additional context. Useful for decoding the 'details' field, if needed.

  details:
    type: any
    properties:
    description: |
      Additional structured data.
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

- `callback`: Optional. If the operation is asynchronous, the handler should invoke this URL once the operation's
  result is available.

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

- `424 Failed Dependency`: Operation completed as `failed` or `canceled`.

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
  If by the end of the wait period the operation is still running, the request should resolve with a 412 status code
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

- `408 Request Timeout`: The server gave up waiting for operation completion. The request may be retried by the caller.

  **Body**: Empty.

- `412 Precondition Failed`: Operation still running.

  When waiting for completion, the caller may re-issue this request to start a new long poll.

  **Body**: Empty.

- `424 Failed Dependency`: Operation completed as `failed` or `canceled`.

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

## Predefined Handler Errors

For compatiblity of this HTTP spec with future transports, when a handler fails a request, it **should** use one of the
following predefined error codes.

| Name                 | Status Code | Description                                                                                                                      |
| -------------------- | ----------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `BAD_REQUEST`        | 400         | The server cannot or will not process the request due to an apparent client error.                                               |
| `UNAUTHENTICATED`    | 401         | The client did not supply valid authentication credentials for this request.                                                     |
| `UNAUTHORIZED`       | 403         | The caller does not have permission to execute the specified operation.                                                          |
| `NOT_FOUND`          | 404         | The requested resource could not be found but may be available in the future. Subsequent requests by the client are permissible. |
| `RESOURCE_EXHAUSTED` | 429         | Some resource has been exhausted, perhaps a per-user quota, or perhaps the entire file system is out of space.                   |
| `INTERNAL`           | 500         | An internal error occured.                                                                                                       |
| `NOT_IMPLEMENTED`    | 501         | The server either does not recognize the request method, or it lacks the ability to fulfill the request.                         |
| `UNAVAILABLE`        | 503         | The service is currently unavailable.                                                                                            |
| `DOWNSTREAM_ERROR`   | 520         | Used by gateways to report that a downstream server has responded with an error.                                                 |
| `DOWNSTREAM_TIMEOUT` | 521         | Used by gateways to report that a request to a downstream server has timed out.                                                  |

## General Purpose Headers

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

1. What potential security concerns should be taken into consideration while implementing this protocol?

Security is not part of this specification, but as this is a thin layer on top of HTTP, standard practices should be
used to secure these APIs. For securing callback URLs, see [Callback URLs > Security](#security).
