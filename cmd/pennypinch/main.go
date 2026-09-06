// Copyright © 2026 The Homeport Team
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/permhoot/pennypinch/internal/config"
	"github.com/permhoot/pennypinch/internal/handler"
	"github.com/permhoot/pennypinch/internal/storage"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dataDir := flag.String("data-dir", "", "data directory (default $EXPENSE_DATA_DIR or ./data)")
	flag.Parse()

	if *dataDir == "" {
		*dataDir = os.Getenv("EXPENSE_DATA_DIR")
	}
	if *dataDir == "" {
		*dataDir = "./data"
	}

	if err := os.MkdirAll(*dataDir, 0o750); err != nil {
		return err
	}

	dbPath := filepath.Join(*dataDir, "expenses.db")
	store, err := storage.OpenSQLite(dbPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	cfgPath := filepath.Join(*dataDir, "config.yaml")
	cfg, err := config.LoadManager(cfgPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := cfg.Close(); err != nil {
			log.Printf("close config: %v", err)
		}
	}()

	srv, err := handler.New(store, cfg)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("pennypinch listening on http://localhost%s (data dir: %s)", *addr, *dataDir)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}

	return nil
}
