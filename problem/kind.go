package problem

import (
	"fmt"
	"strconv"
)

// Kind in a enumeration of kinds of errors.
//
// All [Kind]s of errors here listed follow a HTTP status counterpart
// in the 4xx or 5xx range, the contrary is not true. [Kind]s do not
// necessarely keep the same phrase as the HTTP status counterparts.
//
// [Kind]s of errors in the 4xx range are here called "External" and in
// the 5xx "Internal". [Kind.IsExternal] and [Kind.IsInternal] can be
// used to inspect its category.
//
// [Kind]s can also be directly used as HTTP statuses.
type Kind int

// Enumeration of external errors, i.e., client errors.
const (
	// Bad Request
	Malformed Kind = 400

	// Unauthorized
	Unauthenticated Kind = 401

	// Payment Required
	PaymentRequired Kind = 402

	// Forbidden
	Unauthorized Kind = 403

	// Not Found
	NotFound Kind = 404

	// 405 Method Not Allowed

	// Not Acceptable
	UnsupportedAcceptable Kind = 406

	// 407 Proxy Authentication Required

	// Request Timeout
	Timeout Kind = 408

	// Conflict
	Conflict Kind = 409

	// 410 Gone
	// 411 Length Required

	// Precondition Failed
	PreconditionFailed Kind = 412

	// Content Too Large
	TooLarge Kind = 413

	// 414 URI Too Long

	// Unsupported Media Type
	UnsupportedContentType Kind = 415

	// 416 Range Not Satisfiable
	// 417 Expectation Failed
	// "418 I'm a teapot"
	// 421 Misdirected Request

	// Unprocessable Content
	SemanticalError Kind = 422

	// 423 Locked
	// 424 Failed Dependency
	// 425 Too Early
	// 426 Upgrade Required

	// Precondition Required
	LostUpdate Kind = 428

	// Too Many Requests
	TooManyRequests Kind = 429

	// 431 Request Header Fields Too Large
	// 451 Unavailable For Legal Reasons
)

// Enumeration of external errors, i.e., server errors.
const (
	// Internal Server Error
	UnexpectedError Kind = 500

	// Not Implemented
	Unimplemented Kind = 501

	// Bad Gateway
	BadGateway Kind = 502

	// Service Unavailable
	Unavailable Kind = 503

	// Gateway Timeout
	GatewayTimeout Kind = 504

	// 505 HTTP Version Not Supported
	// 506 Variant Also Negotiates

	// Insufficient Storage
	InsufficientStorage Kind = 507

	// 508 Loop Detected
	// 510 Not Extended
	// 511 Network Authentication Required
)

// IsExternal identifies whether the kind falls under the external
// category, i.e., caused by a client.
func (k Kind) IsExternal() bool {
	return 400 <= k && k <= 499
}

// IsExternal identifies whether the kind falls under the external
// category, i.e., not caused by the client, and usually outside of
// its control.
func (k Kind) IsInternal() bool {
	return 500 <= k && k <= 599
}

var kindStrings = map[Kind]string{
	Malformed:              "malformed",
	Unauthenticated:        "unauthenticated",
	PaymentRequired:        "payment required",
	Unauthorized:           "unauthorized",
	NotFound:               "not found",
	UnsupportedAcceptable:  "unsupported acceptable",
	Timeout:                "timeout",
	Conflict:               "conflict",
	PreconditionFailed:     "precondition failed",
	TooLarge:               "too large",
	UnsupportedContentType: "unsupported content type",
	SemanticalError:        "semantical error",
	LostUpdate:             "lost update",
	TooManyRequests:        "too many requests",
}

var stringKinds = invert(kindStrings)

// String implements the [fmt.Stringer] interface on the type.
func (k Kind) String() string {
	return kindStrings[k]
}

// MarshalJSON implements the [json.Marshaler] interface on the type.
func (k Kind) MarshalJSON() ([]byte, error) {
	quoted := strconv.Quote(kindStrings[k])
	return []byte(quoted), nil
}

// UnmarshalJSON implements the [json.Unmarshaler] interface on the type.
func (k *Kind) UnmarshalJSON(buf []byte) error {
	unquoted, err := strconv.Unquote(string(buf))
	if err != nil {
		return fmt.Errorf("problem: kind must be a valid JSON string: %w", err)
	}

	*k = stringKinds[unquoted]
	return nil
}

func invert[K, V comparable](m map[K]V) map[V]K {
	nm := make(map[V]K, len(m))
	for k, v := range m {
		nm[v] = k
	}

	return nm
}
