package grpcError

import (
	"time"

	errorBuf "github.com/diki-haryadi/protobuf-template/shared/error/v1"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcErr struct {
	Status    codes.Code        `json:"status,omitempty"`
	Code      int               `json:"code,omitempty"`
	Title     string            `json:"title,omitempty"`
	Msg       string            `json:"msg,omitempty"`
	Details   map[string]string `json:"errorDetail,omitempty"`
	Timestamp time.Time         `json:"timestamp,omitempty"`
}

type GrpcErr interface {
	GetStatus() codes.Code
	SetStatus(status codes.Code) GrpcErr
	GetCode() int
	SetCode(code int) GrpcErr
	GetTitle() string
	SetTitle(title string) GrpcErr
	GetMsg() string
	SetMsg(msg string) GrpcErr
	GetDetails() map[string]string
	SetDetails(details map[string]string) GrpcErr
	GetTimestamp() time.Time
	SetTimestamp(time time.Time) GrpcErr
	Error() string
	ErrBody() error
	ToGrpcResponseErr() error
}

func NewGrpcError(status codes.Code, code int, title string, message string, details map[string]string) GrpcErr {
	grpcErr := &grpcErr{
		Status:    status,
		Code:      code,
		Title:     title,
		Msg:       message,
		Details:   details,
		Timestamp: time.Now(),
	}

	return grpcErr
}

func (ge *grpcErr) ErrBody() error {
	return ge
}

func (ge *grpcErr) Error() string {
	return ge.Msg
}

func (ge *grpcErr) GetStatus() codes.Code {
	return ge.Status
}

func (ge *grpcErr) SetStatus(status codes.Code) GrpcErr {
	ge.Status = status

	return ge
}

func (ge *grpcErr) GetCode() int {
	return ge.Code
}

func (ge *grpcErr) SetCode(code int) GrpcErr {
	ge.Code = code

	return ge
}

func (ge *grpcErr) GetTitle() string {
	return ge.Title
}

func (ge *grpcErr) SetTitle(title string) GrpcErr {
	ge.Title = title

	return ge
}

func (ge *grpcErr) GetMsg() string {
	return ge.Msg
}

func (ge *grpcErr) SetMsg(message string) GrpcErr {
	ge.Msg = message

	return ge
}

func (ge *grpcErr) GetDetails() map[string]string {
	return ge.Details
}

func (ge *grpcErr) SetDetails(detail map[string]string) GrpcErr {
	ge.Details = detail

	return ge
}

func (ge *grpcErr) GetTimestamp() time.Time {
	return ge.Timestamp
}

func (ge *grpcErr) SetTimestamp(time time.Time) GrpcErr {
	ge.Timestamp = time

	return ge
}

func IsGrpcError(err error) bool {
	var grpcErr GrpcErr
	return errors.As(err, &grpcErr)
}

// ToGrpcResponseErr creates a gRPC error response to send grpc engine
func (ge *grpcErr) ToGrpcResponseErr() error {
	st := status.New(ge.Status, ge.Error())
	mappedErr := &errorBuf.CustomError{
		Title:     ge.Title,
		Code:      int64(ge.Code),
		Msg:       ge.Msg,
		Details:   ge.Details,
		Timestamp: ge.Timestamp.Format(time.RFC3339),
	}
	stWithDetails, _ := st.WithDetails(mappedErr)
	return stWithDetails.Err()
}

func ParseExternalGrpcErr(err error) GrpcErr {
	st := status.Convert(err)
	for _, detail := range st.Details() {
		if t, ok := detail.(*errorBuf.CustomError); ok {
			timestamp, _ := time.Parse(time.RFC3339, t.Timestamp)
			return &grpcErr{
				Status:    st.Code(),
				Code:      int(t.Code),
				Title:     t.Title,
				Msg:       t.Msg,
				Details:   t.Details,
				Timestamp: timestamp,
			}
		}
	}
	return nil
}
