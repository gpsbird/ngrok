package server

import (
	log "code.google.com/p/log4go"
	"net"
	"ngrok/conn"
)

const (
	NotAuthorized = `HTTP/1.0 401 Not Authorized
WWW-Authenticate: Basic realm="ngrok"
Content-Length: 23

Authorization required
`
)

/**
 * Listens for new http connections from the public internet
 */
func httpListener(addr *net.TCPAddr) {
	// bind/listen for incoming connections
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}

	log.Info("Listening for public http connections on %v", getTCPPort(listener.Addr()))
	for {
		// accept new public connections
		tcpConn, err := listener.AcceptTCP()

		if err != nil {
			panic(err)
		}

		// handle the new connection asynchronously
		go httpHandler(tcpConn)
	}
}

/**
 * Handles a new http connection from the public internet
 */
func httpHandler(tcpConn net.Conn) {
	// wrap up the connection for logging
	conn := conn.NewHttp(tcpConn, "pub")

	defer conn.Close()
	defer func() {
		// recover from failures
		if r := recover(); r != nil {
			conn.Warn("Failed with error %v", r)
		}
	}()

	// read out the http request
	req, err := conn.ReadRequest()
	if err != nil {
		panic(err)
	}

	// multiplex to find the right backend host
	conn.Debug("Found hostname %s in request", req.Host)
	tunnel := tunnels.Get("http://" + req.Host)
	if tunnel == nil {
		conn.Info("No tunnel found for hostname %s", req.Host)
		return
	}

	// satisfy auth, if necessary
	conn.Debug("From client: %s", req.Header.Get("Authorization"))
	conn.Debug("To match: %s", tunnel.regMsg.HttpAuth)
	if req.Header.Get("Authorization") != tunnel.regMsg.HttpAuth {
		conn.Info("Authentication failed")
		conn.Write([]byte(NotAuthorized))
		return
	}

	tunnel.HandlePublicConnection(conn)
}
