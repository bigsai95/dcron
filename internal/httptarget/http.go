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
	httpMutex sync.RWMutex
)

type Service struct {
	HttpConn *ghc.Client
	ApiPath  string
}

func (s *Service) NewTarget(ctx context.Context, apiUrl string) error {
	u, err := url.ParseRequestURI(apiUrl)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	host := u.Host

	if u.Path == "" {
		u.Path = "/"
	}
	s.ApiPath = u.Path

	httpMutex.RLock()
	conn, ok := httpConn[host]
	httpMutex.RUnlock()

	if ok {
		s.HttpConn = conn
	} else {
		httpMutex.Lock()
		defer httpMutex.Unlock()

		// Recheck to avoid race condition
		if conn, ok := httpConn[host]; ok {
			s.HttpConn = conn
			return nil
		}

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
	if s.HttpConn == nil {
		return nil, fmt.Errorf("HTTP connection is not initialized")
	}

	return s.HttpConn.Get(ctx, s.ApiPath)
}
