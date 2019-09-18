// Package kmsg contains Kafka request and response types and autogenerated
// serialization and deserialization functions.
//
// This package reserves the right to add new fields to struct types as Kafka
// adds new fields over time without bumping the major API version.
package kmsg

import (
	"encoding/binary"

	"github.com/twmb/kgo/kbin"
)

// Request represents a type that can be requested to Kafka.
type Request interface {
	// Key returns the protocol key for this message kind.
	Key() int16
	// MaxVersion returns the maximum protocol version this message
	// supports.
	//
	// This function allows one to implement a client that chooses message
	// versions based off of the max of a message's max version in the
	// client and the broker's max supported version.
	MaxVersion() int16
	// SetVersion sets the version to use for this request and response.
	SetVersion(int16)
	// GetVersion returns the version currently set to use for the request
	// and response.
	GetVersion() int16
	// AppendTo appends this message in wire protocol form to a slice and
	// returns the slice.
	AppendTo([]byte) []byte
	// ResponseKind returns an empty Response that is expected for
	// this message request.
	ResponseKind() Response
}

// AdminRequest represents a request that must be issued to Kafka controllers.
type AdminRequest interface {
	// IsAdminRequest is a method attached to requests that must be
	// issed to Kafka controllers.
	IsAdminRequest()
	Request
}

// GroupCoordinatorRequest represents a request that must be issued to a
// group coordinator.
type GroupCoordinatorRequest interface {
	// IsGroupCoordinatorRequest is a method attached to requests that
	// must be issued to group coordinators.
	IsGroupCoordinatorRequest()
	Request
}

// TxnCoordinatorRequest represents a request that must be issued to a
// transaction coordinator.
type TxnCoordinatorRequest interface {
	// IsTxnCoordinatorRequest is a method attached to requests that
	// must be issued to transaction coordinators.
	IsTxnCoordinatorRequest()
	Request
}

// Response represents a type that Kafka responds with.
type Response interface {
	// ReadFrom parses all of the input slice into the response type.
	//
	// This should return an error if too much or too little data is input.
	ReadFrom([]byte) error
}

// AppendRequest appends a full message request to dst, returning the updated
// slice. This message is the full body that needs to be written to issue a
// Kafka request.
//
// clientID is optional; nil means to not send, whereas empty means the client
// id is the empty string.
func AppendRequest(
	dst []byte,
	r Request,
	correlationID int32,
	clientID *string,
) []byte {
	dst = append(dst, 0, 0, 0, 0) // reserve length
	dst = kbin.AppendInt16(dst, r.Key())
	dst = kbin.AppendInt16(dst, r.GetVersion())
	dst = kbin.AppendInt32(dst, correlationID)
	dst = kbin.AppendNullableString(dst, clientID)
	dst = r.AppendTo(dst)
	kbin.AppendInt32(dst[:0], int32(len(dst[4:])))
	return dst
}

// StringPtr is a helper to return a pointer to a string.
func StringPtr(in string) *string {
	return &in
}

// ReadRecords reads n records from in and returns them, returning
// kerr.ErrNotEnoughData if in does not contain enough data.
func ReadRecords(n int, in []byte) ([]Record, error) {
	rs := make([]Record, n)
	for i := 0; i < n; i++ {
		length, used := kbin.Varint(in)
		total := used + int(length)
		if used == 0 || length < 0 || len(in) < total {
			return nil, kbin.ErrNotEnoughData
		}
		if err := (&rs[i]).ReadFrom(in[:total]); err != nil {
			return nil, err
		}
		in = in[total:]
	}
	return rs, nil
}

// ReadRecordBatches reads as many record batches as possible from in,
// discarding any final trailing record batch. This is intended to be used
// for processing RecordBatches from a FetchResponse, where Kafka, as an
// internal optimization, may include a partial final RecordBatch.
func ReadRecordBatches(in []byte) []RecordBatch {
	var bs []RecordBatch
	for len(in) > 12 {
		length := int32(binary.BigEndian.Uint32(in[8:]))
		length += 12
		if len(in) < int(length) {
			return bs
		}
		var b RecordBatch
		if err := b.ReadFrom(in[:length]); err != nil {
			return bs
		}
		bs = append(bs, b)
		in = in[length:]
	}
	return bs
}

// ReadV1Messages reads as many v1 message sets as possible from
// in, discarding any final trailing message set. This is intended to be used
// for processing v1 MessageSets from a FetchResponse, where Kafka, as an
// internal optimization, may include a partial final MessageSet.
func ReadV1Messages(in []byte) []MessageV1 {
	var ms []MessageV1
	for len(in) > 12 {
		length := int32(binary.BigEndian.Uint32(in[8:]))
		length += 12
		if len(in) < int(length) {
			return ms
		}
		var m MessageV1
		if err := m.ReadFrom(in[:length]); err != nil {
			return ms
		}
		ms = append(ms, m)
		in = in[length:]
	}
	return ms
}

// ReadV0Messages reads as many v0 message sets as possible from
// in, discarding any final trailing message set. This is intended to be used
// for processing v0 MessageSets from a FetchResponse, where Kafka, as an
// internal optimization, may include a partial final MessageSet.
func ReadV0Messages(in []byte) []MessageV0 {
	var ms []MessageV0
	for len(in) > 12 {
		length := int32(binary.BigEndian.Uint32(in[8:]))
		length += 12
		if len(in) < int(length) {
			return ms
		}
		var m MessageV0
		if err := m.ReadFrom(in[:length]); err != nil {
			return ms
		}
		ms = append(ms, m)
		in = in[length:]
	}
	return ms
}
