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
	jsonData, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to encode message: %v", err)
	}

	_, err = c.Conn.Write(jsonData)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

func (c *Client) Start() {
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, _, err := c.Conn.ReadFromUDP(buffer)
			if err != nil {
				fmt.Printf("Error reading from server: %v\n", err)
				continue
			}

			msg, err := common.FromJSON(buffer[:n])
			if err != nil {
				fmt.Printf("Error parsing message: %v\n", err)
				continue
			}

			switch msg.Type {
			case common.TypePing:
				aliveMsg := common.NewAliveMessage(c.Username)
				jsonData, err := aliveMsg.ToJSON()
				if err != nil {
					fmt.Printf("Error creating alive message: %v\n", err)
					continue
				}

				_, err = c.Conn.Write(jsonData)
				if err != nil {
					fmt.Printf("Error sending alive response: %v\n", err)
				}
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
