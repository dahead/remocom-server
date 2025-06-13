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
	AccessCode    string
}

func NewServer(host string, port int, handler MessageHandler, accessCode string) (*Server, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
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
		pingInterval:  30 * time.Second,
		clientTimeout: 90 * time.Second,
		AccessCode:    accessCode,
	}, nil
}

func (s *Server) parseMessage(buffer []byte, n int) (*common.ChatMessage, error) {
	msg, err := common.FromEncryptedJSON(buffer[:n], s.AccessCode)
	if err != nil {
		msg, err = common.FromJSON(buffer[:n])
		if err != nil {
			return nil, err
		}
	}
	return msg, nil
}

func (s *Server) Start() {
	if s.Running {
		return
	}

	s.Running = true
	fmt.Printf("Chat server started on: %v. Access code: %s\n", s.Addr, s.AccessCode)

	go func() {
		for s.Running {
			buffer := make([]byte, 4096)

			// Read message
			n, clientAddr, err := s.Conn.ReadFromUDP(buffer)
			if err != nil {
				fmt.Printf("Error reading from UDP: %v\n", err)
				continue
			}

			// Parse message
			msg, err := s.parseMessage(buffer, n)
			if err != nil {
				fmt.Printf("Error parsing message from %v: %v\n", clientAddr, err)
				continue
			}

			// Check passcode validity if message content contains passcode (in this example protocol)
			if msg.Type == common.TypeAuth {
				s.RegisterClient(clientAddr, msg)
				continue
			}

			switch msg.Type {
			case common.TypeAlive:
				s.TryUpdateClienntActivity(clientAddr, msg)
			case common.TypeChat:
				s.ReceiveClientChatMessage(clientAddr, msg)

			}
		}
	}()

	go s.pingClients()
}

func (s *Server) RegisterClient(clientAddr *net.UDPAddr, msg *common.ChatMessage) {
	/// fmt.Printf("Auth message received: %s: %s\n", clientAddr.String(), msg.Content)
	if msg.Content != s.AccessCode {
		// fmt.Printf("Client %s failed passcode validation\n", clientAddr.String())
		return
	}
	s.clients[clientAddr.String()] = &Client{
		Addr:         clientAddr,
		Username:     msg.Username,
		LastActivity: time.Now(),
	}
	fmt.Printf("Client %s successfully authenticated\n", clientAddr.String())
}

func (s *Server) ReceiveClientChatMessage(clientAddr *net.UDPAddr, msg *common.ChatMessage) {
	// Check if client is known and authorized
	client, exists := s.clients[clientAddr.String()]

	if !exists {
		return
	}

	//
	if s.Handler != nil {
		s.Handler(msg, clientAddr)
	}

	// Update client's last activity and username
	client.LastActivity = time.Now()
	client.Username = msg.Username

	// Broadcast chat message to all authenticated clients
	if err := s.Broadcast(msg); err != nil {
		fmt.Printf("Error broadcasting message: %v\n", err)
	}

}

func (s *Server) TryUpdateClienntActivity(clientAddr *net.UDPAddr, msg *common.ChatMessage) {
	if client, exists := s.clients[clientAddr.String()]; exists {
		client.LastActivity = time.Now()
		client.Username = msg.Username
	} else {
		// New client responding to a ping and sent correct password
		s.clients[clientAddr.String()] = &Client{
			Addr:         clientAddr,
			Username:     msg.Username,
			LastActivity: time.Now(),
		}
		fmt.Printf("New client connected: %s (%s)\n", msg.Username, clientAddr.String())
	}
}

func (s *Server) Broadcast(message *common.ChatMessage) error {
	// Chat-Nachrichten werden verschlüsselt gesendet
	jsonData, err := message.ToEncryptedJSON(s.AccessCode)
	if err != nil {
		return fmt.Errorf("failed to encode and encrypt message: %v", err)
	}

	for _, client := range s.clients {
		_, err = s.Conn.WriteToUDP(jsonData, client.Addr)
		if err != nil {
			return fmt.Errorf("failed to send message to %v: %v", client.Addr, err)
		}
	}

	return nil
}

func (s *Server) pingClients() {
	ticker := time.NewTicker(s.pingInterval)
	defer ticker.Stop()

	for s.Running {
		<-ticker.C

		pingMsg := common.NewPingMessage()
		jsonData, err := pingMsg.ToJSON()
		if err != nil {
			fmt.Printf("Error creating ping message: %v\n", err)
			continue
		}

		now := time.Now()
		for addr, client := range s.clients {
			if now.Sub(client.LastActivity) > s.clientTimeout {
				fmt.Printf("Client timeout: %s (%s)\n", client.Username, addr)
				delete(s.clients, addr)
				continue
			}

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
