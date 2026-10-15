// weather-mcp-server-cn: Chinese-friendly fork of ezh0v/weather-mcp-server.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/Lorcan-L/weather-mcp-server/internal/server"
	"github.com/Lorcan-L/weather-mcp-server/internal/weather"
)

func main() {
	httpAddr := flag.String("http", "", "Serve streamable HTTP on this address (endpoint /mcp), e.g. :8000")
	sseAddr := flag.String("address", "", "Legacy SSE transport address (kept for compatibility)")
	flag.Parse()

	timeout := 10 * time.Second
	om := weather.NewOpenMeteo(timeout)
	deps := server.Deps{Weather: om, Air: om}
	if key := os.Getenv("QWEATHER_API_KEY"); key != "" {
		host := os.Getenv("QWEATHER_API_HOST")
		if host == "" {
			log.Fatal("QWEATHER_API_HOST is required when QWEATHER_API_KEY is set (see QWeather console)")
		}
		deps.Weather = weather.NewQWeather(key, host, timeout)
	}
	s := server.New(deps)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	switch {
	case *httpAddr != "":
		h := mcpserver.NewStreamableHTTPServer(s)
		go func() {
			log.Printf("streamable HTTP MCP on %s/mcp", *httpAddr)
			if err := h.Start(*httpAddr); err != nil {
				log.Fatal(err)
			}
		}()
		<-ctx.Done()
		_ = h.Shutdown(context.Background())
	case *sseAddr != "":
		h := mcpserver.NewSSEServer(s)
		go func() {
			if err := h.Start(*sseAddr); err != nil {
				log.Fatal(err)
			}
		}()
		<-ctx.Done()
		_ = h.Shutdown(context.Background())
	default:
		if err := mcpserver.ServeStdio(s); err != nil {
			log.Fatal(err)
		}
	}
}
