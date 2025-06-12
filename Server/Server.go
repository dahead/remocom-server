package server

import (
	"fmt"
	"net"
	"remocom/Common"
	"time"
)

type MessageHandler func(*common.ChatMessage, *net.UDPAddr)

type Client struct {
	Addr         *net.UDPAddr
	Username     string
	LastActivity time.Time
}

type Server struct {
	Addr          *net.UDPAddr
	Conn          *net.UDPConn
	Handler       MessageHandler
	Running       bool
	clients       map[string]*Client
	pingInterval  time.Duration
	clientTimeout time.Duration
}

func NewServer(port int, handler MessageHandler) (*Server, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %v", port, err)
	}

	return &Server{
		Addr:          addr,
		Conn:          conn,
		Handler:       handler,
		Running:       false,
		clients:       make(map[string]*Client),
		pingInterval:  30 * time.Second, // Ping every 30 seconds
		clientTimeout: 90 * time.Second, // Remove clients after 90 seconds of inactivity
	}, nil
}

func (s *Server) Start() {
	if s.Running {
		return
	}

	s.Running = true
	fmt.Printf("Chat server started on %v\n", s.Addr)

	// Start the message handling goroutine
	go func() {
		for s.Running {
			buffer := make([]byte, 4096)
			n, clientAddr, err := s.Conn.ReadFromUDP(buffer)
			if err != nil {
				fmt.Printf("Error reading from UDP: %v\n", err)
				continue
			}

			msg, err := common.FromJSON(buffer[:n])
			if err != nil {
				fmt.Printf("Error parsing message from %v: %v\n", clientAddr, err)
				continue
			}

			// Handle different message types
			switch msg.Type {
			case common.TypeAlive:
				// Update client's last activity time
				if client, exists := s.clients[clientAddr.String()]; exists {
					client.LastActivity = time.Now()
					client.Username = msg.Username
				} else {
					// New client responding to a ping
					s.clients[clientAddr.String()] = &Client{
						Addr:         clientAddr,
						Username:     msg.Username,
						LastActivity: time.Now(),
					}
					fmt.Printf("New client connected: %s (%s)\n", msg.Username, clientAddr.String())
				}
			case common.TypeChat:
				// Handle regular chat message
				if s.Handler != nil {
					s.Handler(msg, clientAddr)
				}

				// Update or add client
				if client, exists := s.clients[clientAddr.String()]; exists {
					client.LastActivity = time.Now()
					client.Username = msg.Username
				} else {
					s.clients[clientAddr.String()] = &Client{
						Addr:         clientAddr,
						Username:     msg.Username,
						LastActivity: time.Now(),
					}
					fmt.Printf("New client connected: %s (%s)\n", msg.Username, clientAddr.String())
				}

				// Nachricht an alle Clients senden
				if err := s.Broadcast(msg); err != nil {
					fmt.Printf("Fehler beim Broadcast der Nachricht: %v\n", err)
				}

			}
		}
	}()

	// Start the client ping goroutine
	go s.pingClients()
}

func (s *Server) pingClients() {
	ticker := time.NewTicker(s.pingInterval)
	defer ticker.Stop()

	for s.Running {
		<-ticker.C

		// Send ping to all clients
		pingMsg := common.NewPingMessage()
		jsonData, err := pingMsg.ToJSON()
		if err != nil {
			fmt.Printf("Error creating ping message: %v\n", err)
			continue
		}

		// Check for inactive clients and ping active ones
		now := time.Now()
		for addr, client := range s.clients {
			// Remove inactive clients
			if now.Sub(client.LastActivity) > s.clientTimeout {
				fmt.Printf("Client timeout: %s (%s)\n", client.Username, addr)
				delete(s.clients, addr)
				continue
			}

			// Ping active clients
			_, err = s.Conn.WriteToUDP(jsonData, client.Addr)
			if err != nil {
				fmt.Printf("Error pinging client %s: %v\n", addr, err)
			}
		}

		fmt.Printf("Active clients: %d\n", len(s.clients))
	}
}

func (s *Server) Stop() {
	s.Running = false
	if s.Conn != nil {
		s.Conn.Close()
	}
}

func (s *Server) Broadcast(message *common.ChatMessage) error {
	jsonData, err := message.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to encode message: %v", err)
	}

	// send message to all clients
	for _, client := range s.clients {
		_, err = s.Conn.WriteToUDP(jsonData, client.Addr)
		if err != nil {
			return fmt.Errorf("failed to send message to %v: %v", client, err)
		}
	}

	return nil
}
