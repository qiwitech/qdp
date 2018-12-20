package api

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/qiwitech/qdp/proto/gatepb"
	"github.com/qiwitech/qdp/pt"
)

type ClientFunc func(baseurl string) gatepb.ProcessorServiceInterface

type Router struct {
	clientsMu    sync.Mutex
	router       pt.Router
	clients      map[string]gatepb.ProcessorServiceInterface
	client       ClientFunc
	getClient    func(uint64) (gatepb.ProcessorServiceInterface, error)
	routeCall    func(uint64, func(gatepb.ProcessorServiceInterface) (*gatepb.Status, error)) error
	checkReroute func(*gatepb.Status) (bool, error)
}

func NewRouter(router pt.Router, client ClientFunc) *Router {
	r := &Router{
		router:  router,
		clients: make(map[string]gatepb.ProcessorServiceInterface),
		client:  client,
	}
	r.getClient = r.getClientForAccount
	r.routeCall = r.routeAndRetry
	r.checkReroute = r.decodeAndCheckReroute
	return r
}

func (r *Router) getClientForAccount(accID uint64) (gatepb.ProcessorServiceInterface, error) {
	url := r.getURLForAccount(accID)
	if url == "" {
		return nil, errors.Errorf("no gate url for account %d", accID)
	}

	// add client to pool
	r.clientsMu.Lock()
	cl, ok := r.clients[url]
	if !ok {
		cl = r.client(url)
		r.clients[url] = cl
	}
	r.clientsMu.Unlock()

	return cl, nil
}

func (r *Router) decodeAndCheckReroute(st *gatepb.Status) (bool, error) {
	if st == nil {
		return false, errors.New("no status")
	}
	switch gatepb.TransferCode(st.Code) {
	case gatepb.TransferCode_SEE_OTHER:
		rmap, err := getRouteMap(st)
		if err != nil {
			return false, errors.Wrap(err, "api reroute")
		}
		if rmap == nil {
			return false, errors.New("no routing info")
		}
		r.router.SetNodes(rmap.Nodes)
		return true, nil
	}
	return false, nil
}

func (r *Router) routeAndRetry(acc uint64, f func(gatepb.ProcessorServiceInterface) (*gatepb.Status, error)) (err error) {
	var try int
reroute:
	if try == 3 {
		return errors.New("rerouting loop")
	}
	try++

	cl, err := r.getClient(acc)
	if err != nil {
		return errors.Wrap(err, "api route")
	}

	st, err := f(cl)
	if err != nil {
		return err
	}

	if reroute, err := r.checkReroute(st); err != nil {
		return errors.Wrap(err, "api reroute check")
	} else if reroute {
		goto reroute
	}

	return nil
}

func (r *Router) ProcessTransfer(ctx context.Context, req *gatepb.TransferRequest) (*gatepb.TransferResponse, error) {
	var res *gatepb.TransferResponse
	err := r.routeCall(req.Sender, func(cl gatepb.ProcessorServiceInterface) (*gatepb.Status, error) {
		var err error
		res, err = cl.ProcessTransfer(ctx, req)
		if res == nil {
			return nil, err
		}
		return res.Status, err
	})
	return res, err
}

func (r *Router) GetPrevHash(ctx context.Context, req *gatepb.GetPrevHashRequest) (*gatepb.GetPrevHashResponse, error) {
	var res *gatepb.GetPrevHashResponse
	err := r.routeCall(req.Account, func(cl gatepb.ProcessorServiceInterface) (*gatepb.Status, error) {
		var err error
		res, err = cl.GetPrevHash(ctx, req)
		if res == nil {
			return nil, err
		}
		return res.Status, err
	})
	return res, err
}

func (r *Router) GetBalance(ctx context.Context, req *gatepb.GetBalanceRequest) (*gatepb.GetBalanceResponse, error) {
	var res *gatepb.GetBalanceResponse
	err := r.routeCall(req.Account, func(cl gatepb.ProcessorServiceInterface) (*gatepb.Status, error) {
		var err error
		res, err = cl.GetBalance(ctx, req)
		if res == nil {
			return nil, err
		}
		return res.Status, err
	})
	return res, err
}

func (r *Router) UpdateSettings(ctx context.Context, req *gatepb.SettingsRequest) (*gatepb.SettingsResponse, error) {
	var res *gatepb.SettingsResponse
	err := r.routeCall(req.Account, func(cl gatepb.ProcessorServiceInterface) (*gatepb.Status, error) {
		var err error
		res, err = cl.UpdateSettings(ctx, req)
		if res == nil {
			return nil, err
		}
		return res.Status, err
	})
	return res, err
}

func (r *Router) GetLastSettings(ctx context.Context, req *gatepb.GetLastSettingsRequest) (*gatepb.GetLastSettingsResponse, error) {
	var res *gatepb.GetLastSettingsResponse
	err := r.routeCall(req.Account, func(cl gatepb.ProcessorServiceInterface) (*gatepb.Status, error) {
		var err error
		res, err = cl.GetLastSettings(ctx, req)
		if res == nil {
			return nil, err
		}
		return res.Status, err
	})
	return res, err
}

func (r *Router) getURLForAccount(accID uint64) string {
	key := fmt.Sprintf("%d", accID)
	host := r.router.GetHostByKey(key)
	if host == "" {
		return ""
	}

	return host
}

func getRouteMap(st *gatepb.Status) (*gatepb.RouteMap, error) {
	if len(st.Details) == 0 {
		return nil, nil
	}
	d := st.Details[0]

	tp := proto.MessageType(d.TypeUrl).Elem()
	val := reflect.New(tp).Interface()
	if err := proto.Unmarshal(d.Value, val.(proto.Message)); err != nil {
		return nil, errors.Wrap(err, "decode status details")
	}

	rmap, ok := val.(*gatepb.RouteMap)
	if !ok {
		return nil, errors.Errorf("invalid route map type: %s", reflect.TypeOf(val))
	}

	if len(rmap.Nodes) == 0 {
		return nil, errors.New("empty nodes list")
	}

	return rmap, nil
}
