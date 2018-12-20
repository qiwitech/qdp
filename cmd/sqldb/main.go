package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/facebookgo/flagenv"
	_ "github.com/go-sql-driver/mysql"
	"github.com/qiwitech/qdp/proto/plutodbpb"
	"github.com/qiwitech/qdp/proto/pusherpb"
	"github.com/qiwitech/qdp/pusher/remotepusher"
	"github.com/qiwitech/qdp/sqlchain"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/qiwitech/tcprpc"
	"golang.org/x/net/trace"
)

var (
	Version = "HEAD"
	Commit  = "dev"
)

var (
	listen = flag.String("listen", ":38388", "http addr")
	dbauth = flag.String("auth", "", "db auth data (user:pass)")
	dbaddr = flag.String("dbaddr", ":3306", "mysql db address")
	dbname = flag.String("dbname", "plutodb", "db name")
	create = flag.String("createuser", "", "create new user and grant permissions (user:pass)")
	drop   = flag.Bool("drop", false, "drop database before start")
	//	meta   = flag.Bool("meta", false, "enable metadb handler")
)

func init() {
	flagenv.Prefix = "PLUTODB_"
	flagenv.Parse()
}

func main() {
	flag.Parse()

	log.Printf("command: %q", os.Args)

	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) {
		return true, true
	}

	lis, err := net.Listen("tcp", *listen)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Start server on %v. Commit: %s, Version: %s\n", lis.Addr(), Commit, Version)

	server := tcprpc.NewServer()

	db, err := sql.Open("mysql", *dbauth+"@tcp("+*dbaddr+")/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "sql open: %v\n", err)
		os.Exit(1) // docker doesn't restart container if error code != 1
	}

	if *drop {
		_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", *dbname))
		if err != nil {
			fmt.Fprintf(os.Stderr, "sql exec: %v\n", err)
			os.Exit(1) // docker doesn't restart container if error code != 1
		}
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", *dbname))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sql exec: %v\n", err)
		os.Exit(1) // docker doesn't restart container if error code != 1
	}

	_, err = db.Exec(fmt.Sprintf("USE %s", *dbname))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sql exec: %v\n", err)
		os.Exit(1) // docker doesn't restart container if error code != 1
	}

	if *create != "" {
		createuser(db)
		return
	}

	p, err := sqlchain.New(db)
	if err != nil {
		panic(err)
	}

	/*
		if *meta {
			p := metadb.New(sdb)
			svc := metadb.NewMetaDBService(p)
			metadbpb.RegisterMetaDBServiceHandlers(server, "v1/", svc)
		}
	*/

	pusherpb.RegisterPusherServiceHandlers(server, "v1/", remotepusher.NewService(p))
	pusherpb.RegisterSettingsPusherServiceHandlers(server, "v1/", remotepusher.NewSettingsService(p))
	plutodbpb.RegisterPlutoDBServiceHandlers(server, "v1/", p)

	http.Handle("/metrics", promhttp.Handler())
	server.HandleHTTP(http.DefaultServeMux)

	if err := server.Serve(lis); err != nil {
		// skip closing server error
		//	if err != http.ErrServerClosed {
		panic(err)
		//	}
	}
}

func createuser(db *sql.DB) {
	u := strings.SplitN(*create, ":", 2)
	if len(u) != 2 {
		fmt.Printf("expected user in form of \"username:password\", got %q\n", *create)
		return
	}
	_, err := db.Exec(fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", u[0], u[1]))
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	_, err = db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON *.* TO '%s'@'%%'", u[0]))
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	_, err = db.Exec("FLUSH PRIVILEGES")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
}
