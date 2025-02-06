package server

import (
	"context"
	"crypto/tls"
	"math/rand"
	"net/http"
	"ngrok/pkg/conn"
	log "ngrok/pkg/log"
	"ngrok/pkg/msg"
	"ngrok/pkg/server/auth"
	"ngrok/pkg/server/db"
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
	opts      *Options
	listeners map[string]*conn.Listener
)

func NewProxy(pxyConn conn.Conn, regPxy *msg.RegProxy) {
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
func tunnelListener(ctx context.Context, addr string, tlsConfig *tls.Config) {
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
			go handleTunnelConnection(ctx, c)
		}
	}
}

func handleTunnelConnection(ctx context.Context, tunnelConn conn.Conn) {
	// don't crash on panics
	defer func() {
		if r := recover(); r != nil {
			tunnelConn.Info("tunnelListener failed with error %v: %s", r, debug.Stack())
		}
	}()

	// Set an initial read deadline
	tunnelConn.SetReadDeadline(time.Now().Add(connReadTimeout))

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

	// Handle the message type
	switch m := rawMsg.(type) {
	case *msg.Auth:
		NewControl(ctx, tunnelConn, m)

	case *msg.RegProxy:
		NewProxy(tunnelConn, m)

	default:
		tunnelConn.Close()
	}
}

func Main() {
	ctx := context.Background()
	// parse options
	opts = parseArgs()

	// init logging
	log.LogTo(opts.logto, opts.loglevel)

	// seed random number generator
	seed, err := util.RandomSeed()
	if err != nil {
		panic(err)
	}
	rand.NewSource(seed)

	// init tunnel/control registry
	registryCacheFile := os.Getenv("REGISTRY_CACHE_FILE")
	tunnelRegistry = NewTunnelRegistry(registryCacheSize, registryCacheFile)
	controlRegistry = NewControlRegistry()

	// start listeners
	listeners = make(map[string]*conn.Listener)

	// load tls configuration
	tlsConfig, err := LoadTLSConfig(opts.tlsCrt, opts.tlsKey)
	if err != nil {
		panic(err)
	}

	// listen for http
	if opts.httpAddr != "" {
		listeners["http"] = startHttpListener(opts.httpAddr, nil)
	}

	// listen for https
	if opts.httpsAddr != "" {
		listeners["https"] = startHttpListener(opts.httpsAddr, tlsConfig)
	}

	// Connect to DB
	dbconn, err := db.GetConnection()
	if err != nil {
		log.Error("Failed to get database connection: %v", err)
	}
	defer dbconn.Close() // Ensure the connection is closed when the program finishes

	// Pass the db connection to PrepareDB
	if err := db.PrepareDB(dbconn); err != nil {
		log.Error("Failed to prepare database: %v", err)
	}

	// Admin endpoint
	go func() {
		http.HandleFunc("/", auth.HomePage)
		http.HandleFunc("/about", auth.ShowAboutPage)
		http.HandleFunc("/keys", auth.GetAPIKeys)
		http.HandleFunc("/add", auth.AddAPIKey)
		http.HandleFunc("/del", auth.RemoveAPIKey)
		http.HandleFunc("/static/", auth.ServeStaticFiles)

		log.Info("Starting Web Admin endpoint on %s", opts.adminAddr)
		if err := http.ListenAndServe(opts.adminAddr, nil); err != nil {
			log.Error("Failed to start status server: %v", err)
			os.Exit(1)
		}
	}()

	// Health endpoint
	go func() {
		http.HandleFunc("/status", auth.Health)
		log.Info("Starting health endpoint on %s", opts.healthAddr)
		if err := http.ListenAndServe(opts.healthAddr, nil); err != nil {
			log.Error("Failed to start status server: %v", err)
			os.Exit(1)
		}
	}()

	// ngrok clients
	tunnelListener(ctx, opts.tunnelAddr, tlsConfig)

}
