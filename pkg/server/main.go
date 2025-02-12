package server

import (
	"context"
	"crypto/tls"
	"math/rand"
	"net/http"
	"ngrok/pkg/conn"
	"ngrok/pkg/msg"
	"ngrok/pkg/server/auth"
	"ngrok/pkg/server/config"
	log "ngrok/pkg/server/log"
	"ngrok/pkg/util"
	"os"
	"runtime/debug"
	"time"
)

const (
	registryCacheSize uint64        = 1024 * 1024 // 1 MB
	connReadTimeout   time.Duration = 10 * time.Second
)

// GLOBALS
var (
	tunnelRegistry  *TunnelRegistry
	controlRegistry *ControlRegistry

	// XXX: kill these global variables - they're only used in tunnel.go for constructing forwarding URLs
	listeners map[string]*conn.Listener
)

func NewProxy(config *config.Config, pxyConn conn.Conn, regPxy *msg.RegProxy) {
	// fail gracefully if the proxy connection fails to register
	defer func() {
		if r := recover(); r != nil {
			pxyConn.Warn("Failed with error: %v", r)
			pxyConn.Close()
		}
	}()

	// set logging prefix
	pxyConn.SetType("pxy")

	// look up the control connection for this proxy
	pxyConn.Info("Registering new proxy for %s", regPxy.ClientId)
	ctl := controlRegistry.Get(regPxy.ClientId)

	if ctl == nil {
		panic("No client found for identifier: " + regPxy.ClientId)
	}

	ctl.RegisterProxy(pxyConn)
}

// Listen for incoming control and proxy connections
// We listen for incoming control and proxy connections on the same port
// for ease of deployment. The hope is that by running on port 443, using
// TLS and running all connections over the same port, we can bust through
// restrictive firewalls.
func tunnelListener(ctx context.Context, config *config.Config, addr string, tlsConfig *tls.Config) {
	listener, err := conn.Listen(addr, "tun", tlsConfig)
	if err != nil {
		log.Error("Fatal error: failed to start listener on %s: %v", addr, err)
		panic(err)
	}

	log.Info("Listening for control and proxy connections on %s", listener.Addr.String())
	stopCh := make(chan struct{})

	go func() {
		<-ctx.Done() // Wait for shutdown signal
		log.Info("Shutting down tunnel listener on %s", addr)
		go func() {
			for c := range listener.Conns { // close each channel individually
				c.Warn("Server shutting down, closing connection %v", c.RemoteAddr())
				_ = c.Close()
			}
		}()
	}()

	for {
		select {
		case <-stopCh:
			log.Info("Tunnel listener stopped: %s", addr)
			return
		case c, ok := <-listener.Conns:
			if !ok {
				log.Warn("Listener channel closed, shutting down")
				return
			}
			go handleTunnelConnection(ctx, config, c)
		}
	}
}

func handleTunnelConnection(ctx context.Context, config *config.Config, tunnelConn conn.Conn) {
	log.Info("handleTunnelConnection: entered")
	// don't crash on panics
	defer func() {
		if r := recover(); r != nil {
			tunnelConn.Info("tunnelListener failed with error %v: %s", r, debug.Stack())
		}
	}()

	// Set an initial read deadline
	tunnelConn.SetReadDeadline(time.Now().Add(connReadTimeout))

	log.Info("handleTunnelConnection: reading message")
	// Read a message from the tunnel connection
	var rawMsg msg.Message
	var err error
	if rawMsg, err = msg.ReadMsg(tunnelConn); err != nil {
		tunnelConn.Warn("Failed to read message: %v", err)
		tunnelConn.Close()
		return
	}

	// Clear the read deadline (heartbeat will handle dead connections)
	tunnelConn.SetReadDeadline(time.Time{})

	log.Info("handleTunnelConnection: handling message %+v", rawMsg)
	// Read a message from the tunnel connection
	// Handle the message type
	switch m := rawMsg.(type) {
	case *msg.Auth:
		NewControl(ctx, config, tunnelConn, m)

	case *msg.RegProxy:
		NewProxy(config, tunnelConn, m)

	default:
		tunnelConn.Close()
	}
}

func Main() {
	ctx := context.Background()
	// parse options
	config := config.InitConfig()

	servingDomain = config.Domain
	proxyMaxPoolSize = config.ProxyMaxPoolSize

	// init logging
	log.LogTo(config.LogLevel)

	// seed random number generator
	seed, err := util.RandomSeed()
	if err != nil {
		panic(err)
	}
	rand.NewSource(seed)

	// init tunnel/control registry
	tunnelRegistry = NewTunnelRegistry(registryCacheSize, config.RegistryCacheFile)
	controlRegistry = NewControlRegistry()

	// start listeners
	listeners = make(map[string]*conn.Listener)

	// load tls configuration
	tlsConfig, err := LoadTLSConfig(config.TLSCert, config.TLSKey)
	if err != nil {
		panic(err)
	}

	// listen for http
	if config.HttpAddr != "" {
		listeners["http"] = startHttpListener(config.HttpAddr, nil)
	}

	// listen for https
	if config.HttpsAddr != "" {
		listeners["https"] = startHttpListener(config.HttpsAddr, tlsConfig)
	}

	handler := auth.Handler{Config: config}
	if config.AdminAddr != "" {
		// Admin endpoint
		go func() {
			http.HandleFunc("/", handler.HomePage)
			http.HandleFunc("/keys", handler.GetAPIKeys)
			http.HandleFunc("/add", handler.AddAPIKey)
			http.HandleFunc("/del", handler.RemoveAPIKey)
			http.HandleFunc("/static/", handler.ServeStaticFiles)

			log.Info("Starting Web Admin endpoint on %s", config.AdminAddr)
			if err := http.ListenAndServe(config.AdminAddr, nil); err != nil {
				log.Error("Failed to start status server: %v", err)
				os.Exit(1)
			}
		}()
	}

	if config.HealthAddr != "" {
		go func() {
			http.HandleFunc("/status", handler.Health)
			log.Info("Starting health endpoint on %s", config.HealthAddr)
			if err := http.ListenAndServe(config.HealthAddr, nil); err != nil {
				log.Error("Failed to start status server: %v", err)
				os.Exit(1)
			}
		}()
	}

	// ngrok clients
	tunnelListener(ctx, config, config.TunnelAddr, tlsConfig)

}
