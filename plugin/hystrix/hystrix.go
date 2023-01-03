package hystrix

import (
	"context"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/logger"
)

type clientWrapper struct {
	client.Client
}

// Call 熔断逻辑
func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	return hystrix.Do(req.Service()+"."+req.Endpoint(), func() error {
		logger.Info(req.Service() + "." + req.Endpoint())
		return c.Client.Call(ctx, req, rsp, opts...)
	}, func(e error) error {
		fmt.Println(req.Service() + "." + req.Endpoint() + "的熔断逻辑")
		return e
	})
}

func NewClientHystrixWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &clientWrapper{}
	}
}
