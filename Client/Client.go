package client

import (
	"fmt"
	"net"
	"remocom/Common"
)

type Client struct {
	ServerAddr *net.UDPAddr
	Conn       *net.UDPConn
	Username   string
	AccessCode string
}

func NewClient(serverHost string, serverPort int, username string) (*Client, error) {
	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", serverHost, serverPort))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}

	return &Client{
		ServerAddr: serverAddr,
		Conn:       conn,
		Username:   username,
	}, nil
}

func (c *Client) SendMessage(content string) error {
	msg := common.NewChatMessage(c.Username, content)

	var jsonData []byte
	var err error

	// Verschlüsselt senden wenn AccessCode verfügbar
	if c.AccessCode != "" {
		jsonData, err = msg.ToEncryptedJSON(c.AccessCode)
		if err != nil {
			return fmt.Errorf("failed to encode and encrypt message: %v", err)
		}
	} else {
		jsonData, err = msg.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to encode message: %v", err)
		}
	}

	_, err = c.Conn.Write(jsonData)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

func (c *Client) Authenticate(username string, accessCode string) error {
	c.AccessCode = accessCode
	authMsg := common.NewAuthenticateMessage(username, accessCode)
	authMsg.Content = accessCode
	authJson, err := authMsg.ToJSON()
	if err != nil {
		return fmt.Errorf("Fehler beim Erstellen der Auth-Nachricht: %v", err)
	}
	_, err = c.Conn.Write(authJson)
	if err != nil {
		return fmt.Errorf("Fehler beim Senden der Auth-Nachricht: %v", err)
	}
	return nil

}

func (c *Client) SendPing() error {
	pingMsg := common.NewPingMessage()
	pingJson, err := pingMsg.ToJSON()
	if err != nil {
		return fmt.Errorf("Fehler beim Erstellen der Ping-Nachricht: %v\n", err)
	}
	_, err = c.Conn.Write(pingJson)
	if err != nil {
		return fmt.Errorf("Fehler beim Senden der Ping-Nachricht: %v\n", err)
	}
	return nil
}

func (c *Client) SendAlive() error {
	// Alive-Nachrichten bleiben unverschlüsselt
	aliveMsg := common.NewAliveMessage(c.Username)
	aliveJson, err := aliveMsg.ToJSON()
	if err != nil {
		return fmt.Errorf("Fehler beim Erstellen der Alive-Nachricht: %v\n", err)

	}
	_, err = c.Conn.Write(aliveJson)
	if err != nil {
		return fmt.Errorf("Fehler beim Senden der Alive-Nachricht: %v\n", err)
	}
	return nil
}

func (c *Client) Start() {
	// nicht blockierend
	go func() {
		buffer := make([]byte, 4096)

		// Endlos Schleife...
		for {
			n, _, err := c.Conn.ReadFromUDP(buffer)
			if err != nil {
				// fmt.Printf("Error reading from server: %v\n", err)
				continue
			}

			var msg *common.ChatMessage

			// Versuche zuerst verschlüsselte Nachricht (falls AccessCode vorhanden)
			if c.AccessCode != "" {
				msg, err = common.FromEncryptedJSON(buffer[:n], c.AccessCode)
			}

			// Falls Entschlüsselung fehlschlägt oder kein AccessCode, versuche unverschlüsselt
			if err != nil || c.AccessCode == "" {
				msg, err = common.FromJSON(buffer[:n])
				if err != nil {
					continue
				}
			}

			switch msg.Type {
			case common.TypePing:
				c.SendPing()
			case common.TypeChat:
				fmt.Printf("[%s] %s: %s\n", msg.Timestamp.Format("15:04:05"), msg.Username, msg.Content)
			}
		}
	}()
}

func (c *Client) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}
}
