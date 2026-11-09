// Command seed-session creates one session in a product home and shuts its
// daemon down again, so a later window launch opens that home with a session in
// its catalog. It is an evidence helper only: it calls the public protocol
// through the supervisor and reads no daemon storage.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
)

func main() {
	home := flag.String("home", "", "product home to seed (required)")
	server := flag.String("server", "", "dim-app-server binary (default: supervisor lookup)")
	cwd := flag.String("cwd", "", "working directory for the new session (default: current)")
	flag.Parse()

	if *home == "" {
		fmt.Fprintln(os.Stderr, "seed-session: -home is required")
		os.Exit(2)
	}
	if *cwd == "" {
		*cwd, _ = os.Getwd()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sup, err := supervisor.New(supervisor.Config{
		ProductHome: *home,
		ServerPath:  *server,
		MockModel:   true,
		ClientName:  "dimagent-seed-session",
		ListenWait:  supervisor.ListenWait{Timeout: 30 * time.Second, Interval: 50 * time.Millisecond},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed-session: supervisor: %v\n", err)
		os.Exit(1)
	}

	conn, err := sup.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed-session: start daemon: %v\n", err)
		os.Exit(1)
	}
	result, err := appserver.Create(ctx, conn.Client(), appserver.CreateParams{Cwd: *cwd})
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed-session: session.create: %v\n", err)
		_, _ = sup.Close(context.Background())
		os.Exit(1)
	}

	closeCtx, closeCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer closeCancel()
	if _, err := sup.Close(closeCtx); err != nil {
		fmt.Fprintf(os.Stderr, "seed-session: close daemon: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.SessionID)
}
