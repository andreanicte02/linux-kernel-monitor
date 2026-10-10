// Command kmon serves the kernel module's data as a JSON API and dashboard,
// or prints a one-off snapshot.
//
//	kmon serve [-addr :8080] [-web ./web]
//	kmon snapshot > snapshot.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/andreanicte02/linux-kernel-monitor/kmon"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kmon serve [-addr :8080] [-web ./web] | kmon snapshot")
		os.Exit(2)
	}
	r := kmon.NewReader()

	switch os.Args[1] {
	case "snapshot":
		s, err := r.Snapshot()
		if err != nil {
			log.Fatal(err)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(s); err != nil {
			log.Fatal(err)
		}

	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		addr := fs.String("addr", ":8080", "listen address")
		web := fs.String("web", "web", "directory with the dashboard files")
		fs.Parse(os.Args[2:])
		if !r.Loaded() {
			log.Fatal("kmon: /proc/kmon_procs not found; load the module first (cd module && make load)")
		}
		srv := &http.Server{Addr: *addr, Handler: routes(r, *web), ReadHeaderTimeout: 5 * time.Second}
		log.Printf("dashboard on http://localhost%s", *addr)
		log.Fatal(srv.ListenAndServe())

	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
}

// routes is read-only on purpose: the original project also exposed
// GET /kill/{pid} without authentication, which was removed.
func routes(r *kmon.Reader, webDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/snapshot", func(w http.ResponseWriter, _ *http.Request) {
		s, err := r.Snapshot()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(s)
	})
	mux.Handle("GET /", http.FileServer(http.Dir(webDir)))
	return mux
}
