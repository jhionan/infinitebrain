// Package ping implements the PingService connect-go handler.
package ping

import (
	"context"

	"connectrpc.com/connect"
	pingv1 "github.com/rian/infinite_brain/api/gen/ping/v1"
	"github.com/rian/infinite_brain/api/gen/ping/v1/pingv1connect"
)

// Handler implements pingv1connect.PingServiceHandler.
type Handler struct{}

// NewHandler returns a new PingService handler.
func NewHandler() pingv1connect.PingServiceHandler {
	return &Handler{}
}

// Ping echoes the request message back in the response.
func (h *Handler) Ping(_ context.Context, req *connect.Request[pingv1.PingRequest]) (*connect.Response[pingv1.PingResponse], error) {
	return connect.NewResponse(&pingv1.PingResponse{
		Message: req.Msg.Message,
	}), nil
}
