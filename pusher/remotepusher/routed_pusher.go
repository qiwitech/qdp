package remotepusher

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/pkg/errors"

	"github.com/qiwitech/qdp/pt"
)

type RoutedPusher struct {
	clientsMu sync.Mutex
	router    pt.Router
	clients   map[string]pt.Pusher
	client    func(baseurl string) pt.Pusher
}

// TODO(outself): add SetClient
func NewRoutedPusher(router pt.Router) *RoutedPusher {
	return &RoutedPusher{
		router:  router,
		clients: make(map[string]pt.Pusher),
		client:  NewHTTPClient,
	}
}

func (r *RoutedPusher) Push(ctx context.Context, txns []pt.Txn) error {
	var g errgroup.Group

	for _, txn := range txns {
		txn := txn
		g.Go(func() error {
			url, _ := r.getRemoteURLForAccount(txn.Receiver)
			//	if self {
			//		return nil
			//	}

			if url == "" {
				return errors.Errorf("no remote push url for account %d", txn.Receiver)
			}

			// add client to pool
			r.clientsMu.Lock()
			cl, ok := r.clients[url]
			if !ok {
				cl = r.client(url)
				r.clients[url] = cl
			}
			r.clientsMu.Unlock()

			return cl.Push(ctx, []pt.Txn{txn})
		})
	}

	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "routed push failed")
	}

	return nil
}

func (r *RoutedPusher) getRemoteURLForAccount(accID pt.AccID) (string, bool) {
	key := fmt.Sprintf("%d", accID)
	host := r.router.GetHostByKey(key)
	if host == "" {
		return "", false
	}

	return host, r.router.IsSelf(host)
}
