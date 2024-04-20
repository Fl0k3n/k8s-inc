package connector

import (
	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// see https://developers.google.com/actions-center/reference/grpc-api/status_codes

type RpcError struct {
	RpcMsg string
}

type EntryExistsError struct {
	RpcError
}

type InvalidEntry struct {
	RpcError
}

func (e *RpcError) Error() string {
	return e.RpcMsg
}

// TODO
func IsNetworkError(err error) bool {
	return false
}

func unwrapP4Error(err error) (*p4_v1.Error, bool) {
	if err == nil {
		return nil, false
	}
	statusErr, ok := status.FromError(err)	
	if !ok {
		return nil, false
	}
	if statusErr.Details() != nil && len(statusErr.Details()) > 0 {
		firstDetail := statusErr.Details()[0]
		p4e, ok := firstDetail.(*p4_v1.Error)
		return p4e, ok	
	}
	return nil, false
}

func checkBasedOnCode(err error, expectedCode grpc_codes.Code) bool {
	if p4e, ok := unwrapP4Error(err); ok {
		return p4e.CanonicalCode == int32(expectedCode)
	}
	return false
}

func checkBasedOnCodeIn(err error, codes ...grpc_codes.Code) bool {
	if p4e, ok := unwrapP4Error(err); ok {
		for _, code := range codes {
			if p4e.CanonicalCode == int32(code) {
				return true
			}
		}
	}
	return false
}

func IsEntryExistsError(err error) bool {
	return checkBasedOnCode(err, grpc_codes.AlreadyExists)
}

func IsInvalidEntryError(err error) bool {
	return checkBasedOnCode(err, grpc_codes.InvalidArgument)
}

func IsEntryNotFoundError(err error) bool {
	return checkBasedOnCode(err, grpc_codes.NotFound)
}

func AsRpcError(err error) (re RpcError, ok bool) {
	p4e, ok := unwrapP4Error(err)
	if !ok {
		return RpcError{}, false
	}
	return RpcError{RpcMsg: p4e.Message}, true
}
