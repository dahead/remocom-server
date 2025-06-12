package main

import (
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

func startServer(host string, port int) {
	chatServer, err := server.NewServer(port, func(msg *common.ChatMessage, addr *net.UDPAddr) {
		fmt.Printf("[%s] %s: %s\n", msg.Timestamp.Format("15:04:05"), msg.Username, msg.Content)
	})
	if err != nil {
		fmt.Printf("Fehler beim Erstellen des Servers: %v\n", err)
		return
	}

	fmt.Printf("Server läuft auf %s:%d\n", host, port)
	chatServer.Start()
	defer chatServer.Stop()

	// Server bleibt aktiv, bis Strg+C gedrückt wird
	select {}
}

func startClient(host string, port int) {
	fmt.Print("Bitte geben Sie Ihren Benutzernamen ein: ")
	var username string
	if _, err := fmt.Scanln(&username); err != nil {
		fmt.Printf("Fehler beim Lesen des Benutzernamens: %v\n", err)
		return
	}
	if username == "" {
		fmt.Println("Benutzername darf nicht leer sein")
		return
	}
	chatClient, err := client.NewClient(host, port, username)
	if err != nil {
		fmt.Printf("Fehler beim Starten des Clients: %v\n", err)
		return
	}
	defer chatClient.Close()

	// Start the client listener
	chatClient.Start()

	fmt.Println("Client gestartet. Geben Sie Nachrichten ein (exit zum Beenden):")

	for {
		var input string
		fmt.Print("> ")
		_, err = fmt.Scanln(&input)
		if err != nil {
			break
		}
		if input == "exit" {
			break
		}
		if strings.TrimSpace(input) != "" {
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
		startServer(host, port)
		return
	}

	switch args[1] {
	case "server":
		host := "localhost"
		port := randomPort()
		if len(args) > 2 {
			h, p, err := parseHostPort(args[2])
			if err != nil {
				fmt.Println(err)
				return
			}
			host = h
			port = p
		}
		startServer(host, port)
	case "client":
		if len(args) != 3 {
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
		fmt.Println("  ./rcs server IP:PORT       # Startet Server auf angegebener Adresse")
		fmt.Println("  ./rcs client IP:PORT       # Startet Client, verbindet zu Adresse")
	}
}
