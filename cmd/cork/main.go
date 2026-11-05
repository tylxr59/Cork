// Command cork runs the Cork server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tylxr59/cork/internal/server"
	"github.com/tylxr59/cork/web"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			os.Exit(healthcheck())
		case "version", "--version", "-v":
			fmt.Println("cork", version)
			return
		}
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envBool(key string) bool {
	b, _ := strconv.ParseBool(env(key, "false"))
	return b
}

func addr() string { return env("CORK_ADDR", ":8080") }

func run() error {
	maxMB, err := strconv.Atoi(env("CORK_MAX_UPLOAD_MB", "25"))
	if err != nil || maxMB <= 0 {
		return errors.New("CORK_MAX_UPLOAD_MB must be a positive number")
	}
	webFS, err := web.FS()
	if err != nil {
		slog.Warn("frontend not embedded; serving API only", "err", err)
	}
	srv, err := server.New(server.Config{
		DataDir:           env("CORK_DATA_DIR", "./data"),
		MaxUploadBytes:    int64(maxMB) << 20,
		TrustProxy:        envBool("CORK_TRUST_PROXY"),
		AllowPrivateFetch: envBool("CORK_ALLOW_PRIVATE_FETCH"),
		AdminUsername:     env("CORK_ADMIN_USERNAME", ""),
		AdminPassword:     os.Getenv("CORK_ADMIN_PASSWORD"),
		Web:               webFS,
	})
	if err != nil {
		return err
	}
	defer srv.Close()

	httpSrv := &http.Server{
		Addr:              addr(),
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpSrv.Shutdown(shutdown)
	}()

	slog.Info("cork listening", "addr", httpSrv.Addr, "version", version)
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// healthcheck lets the distroless container check itself without curl.
func healthcheck() int {
	port := addr()
	if strings.HasPrefix(port, ":") {
		port = "127.0.0.1" + port
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + port + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
