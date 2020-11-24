// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqgrpc

import (
	"context"

	"github.com/sqreen/go-agent/internal"
	grpc_protection "github.com/sqreen/go-agent/internal/protection/http/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func init() {
	internal.Start()
}

func UnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	root, cancel := internal.NewRootHTTPProtectionContext(ctx)
	if ctx == nil {
		return handler(ctx, req)
	}
	defer cancel()

	p := grpc_protection.NewUnaryRPCProtectionContext(root, req, info)

	var (
		resp interface{}
		err  error
	)
	defer func() {
		p.Close(newObservedResponse(err))
	}()

	if err := p.Before(); err != nil {
		return nil, err
	}

	resp, err = handler(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := p.After(); err != nil {
		return nil, err
	}
	return resp, err
}

// response observed by the response writer
type observedResponse struct {
	status int
}

func newObservedResponse(err error) observedResponse {
	return observedResponse{
		status: int(status.Code(err)),
	}
}

func (r observedResponse) Status() int {
	return r.status
}

func (r observedResponse) ContentType() string {
	return ""
}

func (r observedResponse) ContentLength() int64 {
	return 0
}
