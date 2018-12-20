package router

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/go-chi/render"
	"github.com/pkg/errors"
	"github.com/pressly/chi"
	"golang.org/x/sync/errgroup"
)

type UpdatableRouter interface {
	Nodes() []string
	SetNodes([]string)

	Self() string
	SetSelf(string)
}

type HTTPData struct {
	Self  string
	Nodes []string
}

func Handler(r UpdatableRouter) http.Handler {
	e := chi.NewRouter()

	e.Get("/cfg/router", func(w http.ResponseWriter, req *http.Request) {
		d := HTTPData{
			Self:  r.Self(),
			Nodes: r.Nodes(),
		}

		render.JSON(w, req, d)
	})
	e.Post("/cfg/router", func(w http.ResponseWriter, req *http.Request) {
		var d HTTPData

		err := render.DecodeJSON(req.Body, &d)
		if err != nil {
			render.JSON(w, req, map[string]string{"error": err.Error()})
			return
		}

		r.SetSelf(d.Self)
		r.SetNodes(d.Nodes)

		d = HTTPData{
			Self:  r.Self(),
			Nodes: r.Nodes(),
		}

		render.JSON(w, req, d)
	})
	e.Get("/cfg/router/check/{srv}", func(w http.ResponseWriter, req *http.Request) {
		srv := chi.URLParam(req, "srv")

		err := UpdateRouter(r, srv)
		if err != nil {
			render.JSON(w, req, map[string]string{"error": err.Error()})
			return
		}

		d := HTTPData{
			Self:  r.Self(),
			Nodes: r.Nodes(),
		}

		render.JSON(w, req, d)
	})

	return e
}

func UpdateRouter(r UpdatableRouter, srv string) error {
	_, p, err := net.SplitHostPort(srv)
	if err != nil {
		return errors.Wrap(err, "split host port")
	}

	me, addrs, err := CheckService(srv)
	if err != nil {
		return errors.Wrap(err, "check service")
	}

	sort.Strings(addrs)

	if me != "" {
		me = me + ":" + p
	}

	nodes := make([]string, len(addrs))
	for i, addr := range addrs {
		nodes[i] = fmt.Sprintf("%s:%s", addr, p)
	}

	r.SetSelf(me)
	r.SetNodes(nodes)

	return nil
}

func CheckService(srv string) (string, []string, error) {
	host, port, err := net.SplitHostPort(srv)
	if err != nil {
		return "", nil, errors.Wrap(err, "split service")
	}

	me, err := os.Hostname()
	if err != nil {
		return "", nil, errors.Wrap(err, "hostname")
	}

	addrs, err := net.LookupHost(host)
	log.Printf("for service %v found addresses %v", srv, addrs)
	if err != nil {
		return "", nil, errors.Wrap(err, "lookup")
	}

	g, _ := errgroup.WithContext(context.TODO())
	res := make(chan string, 1)

	for _, addr := range addrs {
		addr := addr

		g.Go(func() error {
			u := url.URL{Scheme: "http", Host: fmt.Sprintf("%v:%v", addr, port), Path: "/hostname"}
			// TODO(nik): use errgroup context
			resp, err := http.Get(u.String())
			if err != nil {
				return errors.Wrap(err, "get hostname")
			}

			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrap(err, "read hostname response")
			}
			err = resp.Body.Close()
			if err != nil {
				return errors.Wrap(err, "close response body")
			}

			str := strings.TrimSpace(string(data))

			if str == me {
				res <- addr
			}

			return nil
		})
	}

	err = g.Wait()
	if err != nil {
		return "", nil, errors.Wrap(err, "group")
	}

	select {
	case me = <-res:
	default:
		me = ""
	}

	return me, addrs, nil
}
