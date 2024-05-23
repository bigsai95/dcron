package httptarget

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	ghc "github.com/bozd4g/go-http-client"
)

type ITarget interface {
	NewTarget(ctx context.Context, apiUrl string) error
	GetResponse(ctx context.Context) (*ghc.Response, error)
}

var (
	httpConn  = make(map[string]*ghc.Client)
	httpMutex sync.Mutex
)

type Service struct {
	HttpConn *ghc.Client
	ApiPath  string
}

func (s *Service) NewTarget(ctx context.Context, apiUrl string) error {
	u, err := url.ParseRequestURI(apiUrl)
	if err != nil {
		return err
	}

	host := u.Host

	if u.Path == "" {
		u.Path = "/"
	}
	s.ApiPath = u.Path

	httpMutex.Lock()
	conn, ok := httpConn[host]
	httpMutex.Unlock()

	if ok {
		s.HttpConn = conn
	} else {
		httpMutex.Lock()
		defer httpMutex.Unlock()

		opts := []ghc.ClientOption{
			ghc.WithDefaultHeaders(),
			ghc.WithTimeout(time.Second * 20),
		}

		baseUrl := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		client := ghc.New(baseUrl, opts...)

		s.HttpConn = client
		httpConn[host] = client
	}

	return nil
}

func (s *Service) GetResponse(ctx context.Context) (*ghc.Response, error) {
	return s.HttpConn.Get(ctx, s.ApiPath)
}
