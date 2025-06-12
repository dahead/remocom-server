package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"remocom/Client"
	"remocom/Common"
	"remocom/Server"
)

const defaultAccesscode = "123456"

func randomPort() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(65535-1024) + 1024
}

func parseHostPort(arg string) (string, int, error) {
	parts := strings.Split(arg, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("ungültiges Format, erwartet IP:PORT")
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("ungültiger Port: %v", err)
	}
	return parts[0], port, nil
}

func startServer(host string, port int, accessHashcode string) {
	chatServer, err := server.NewServer(port, func(msg *common.ChatMessage, addr *net.UDPAddr) {
		fmt.Printf("[%s] %s: %s\n", msg.Timestamp.Format("15:04:05"), msg.Username, msg.Content)
	}, accessHashcode)
	if err != nil {
		fmt.Printf("Fehler beim Erstellen des Servers: %v\n", err)
		return
	}

	fmt.Printf("Server running on %s:%d\n", host, port)
	chatServer.Start()
	defer chatServer.Stop()

	select {}
}

func startClient(host string, port int) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Bitte geben Sie Ihren Benutzernamen ein: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Fehler beim Lesen des Benutzernamens: %v\n", err)
		return
	}
	username = strings.TrimSpace(username)
	if username == "" {
		fmt.Println("Benutzername darf nicht leer sein")
		return
	}

	fmt.Print("Bitte geben Sie den Zugangscode ein: ")
	accessCode, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Fehler beim Lesen des Zugangscodes: %v\n", err)
		return
	}
	accessCode = strings.TrimSpace(accessCode)

	chatClient, err := client.NewClient(host, port, username)
	if err != nil {
		fmt.Printf("Fehler beim Starten des Clients: %v\n", err)
		return
	}
	defer chatClient.Close()

	// Send auth message with access code as Content to authenticate
	authMsg := common.NewAuthenticateMessage(username, accessCode)
	authMsg.Content = accessCode
	authJson, err := authMsg.ToJSON()
	if err != nil {
		fmt.Printf("Fehler beim Erstellen der Auth-Nachricht: %v\n", err)
		return
	}
	_, err = chatClient.Conn.Write(authJson)
	if err != nil {
		fmt.Printf("Fehler beim Senden der Auth-Nachricht: %v\n", err)
		return
	}

	chatClient.Start()

	// Todo: when hash access check failed, quit.
	// Or is it better to not know a failed connection?

	fmt.Println("Client gestartet. Geben Sie Nachrichten ein (exit zum Beenden):")

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		input = strings.TrimSpace(input)
		if input == "exit" {
			break
		}
		if input != "" {
			err = chatClient.SendMessage(input)
			if err != nil {
				fmt.Printf("Fehler beim Senden der Nachricht: %v\n", err)
			}
		}
	}
}

func main() {
	args := os.Args
	if len(args) == 1 {
		host := "localhost"
		port := randomPort()
		// Default empty access code on local start
		startServer(host, port, defaultAccesscode)
		return
	}

	switch args[1] {
	case "server":
		host := "localhost"
		port := randomPort()
		accessCode := defaultAccesscode
		if len(args) > 2 {
			h, p, err := parseHostPort(args[2])
			if err != nil {
				fmt.Println(err)
				return
			}
			host = h
			port = p

			if len(args) > 3 {
				accessCode = args[3]
			}
		}
		startServer(host, port, accessCode)
	case "client":
		if len(args) < 3 {
			fmt.Println("Bitte geben Sie die Zieladresse als IP:PORT an. Beispiel: ./app client localhost:44366")
			return
		}
		host, port, err := parseHostPort(args[2])
		if err != nil {
			fmt.Println(err)
			return
		}
		startClient(host, port)
	default:
		fmt.Println("Unbekannter Modus. Verwendung:")
		fmt.Println("  ./rcs                      # Startet Server auf zufälligem Port")
		fmt.Println("  ./rcs server               # Startet Server auf zufälligem Port")
		fmt.Println("  ./rcs server IP:PORT CODE  # Startet Server mit Zugangscode")
		fmt.Println("  ./rcs client IP:PORT       # Startet Client, verbindet zu Adresse")
	}
}
