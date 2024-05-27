// ================================================================================
// Customised SSE broker to handle our chat messages
// ================================================================================

package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/benc-uk/go-rest-api/pkg/sse"
	"github.com/google/uuid"
)

const serverUsername = "💻 Server Message"
const maxMsgsReloaded = 5
const URLRegex = `^(http:\/\/www\.|https:\/\/www\.|http:\/\/|https:\/\/|\/|\/\/)?[A-z0-9_-]*?[:]?[A-z0-9_-]*?[@]?[A-z0-9]+([\-\.]{1}[a-z0-9]+)*\.[a-z]{2,5}(:[0-9]{1,5})?(\/.*)?$`

var re = regexp.MustCompile(URLRegex)

// ChatMessage is the data structure used for chats & system messages
type ChatMessage struct {
	Username  string // Username of the sender
	Message   string // Message body
	Timestamp string
	System    bool // Is this a special system message?
	FromDB    bool
}

func initChat(db *sql.DB, renderer HTMLRenderer) sse.Broker[ChatMessage] {
	// The broker for `ChatMessage` data type
	broker := sse.NewBroker[ChatMessage]()

	// Handle users joining the chat
	broker.ClientConnectedHandler = func(clientID string) {
		broker.SendToGroup("*", ChatMessage{
			Username: serverUsername,
			Message:  fmt.Sprintf("User '%s' has joined the chat 💬", clientID),
			System:   true,
		})

		broker.SendToGroup("*", ChatMessage{
			Username: "",
			Message:  fmt.Sprintf("There are %d users online", broker.GetClientCount()),
			System:   true,
		})

		// Send last 50 messages from store
		msgs := fetchMessages(db, maxMsgsReloaded)
		for _, msg := range msgs {
			broker.SendToClient(clientID, ChatMessage{
				Username:  msg.Username,
				Message:   msg.Message,
				Timestamp: msg.Timestamp,
				FromDB:    true,
			})
		}
	}

	// Handle users leaving the chat
	broker.ClientDisconnectedHandler = func(clientID string) {
		broker.SendToGroup("*", ChatMessage{
			Username: serverUsername,
			Message:  fmt.Sprintf("User '%s' has left the chat 👋", clientID),
			System:   true,
		})

		broker.SendToGroup("*", ChatMessage{
			Username: "",
			Message:  fmt.Sprintf("There are %d users online", broker.GetClientCount()),
			System:   true,
		})
	}

	// Handle chat & system messages and format them for SSE
	broker.MessageAdapter = func(msg ChatMessage, clientID string) sse.SSE {
		sse := sse.SSE{
			Event: "chat",
			Data:  "",
			ID:    uuid.New().String(),
		}

		preview, isLink, err := fetchMetadata(msg.Message)
		if err != nil {
			fmt.Printf("error fetchMetadata %v", err)
		}
		if isLink && preview.Image != "" {
			msg.Message = fmt.Sprintf(`<div class="imgcont">
			<a href="%s"><img class="linkimg" src="%s"></a>
			<p>%s</p>
			</div>`, msg.Message, preview.Image, preview.Title)
		}
		// Render the message using HTML template
		msgHTML, err := renderer.RenderToString("message", map[string]any{
			"username": msg.Username,
			"message":  template.HTML(msg.Message), // nolint:gosec
			"time":     msg.Timestamp,
			"isSelf":   clientID == msg.Username,
			"isServer": msg.System || msg.Username == serverUsername,
		})
		if err != nil {
			fmt.Printf("Error in render to string: %v", err)
		}
		// Write the HTML response, but we need to strip out newlines from the template for SSE
		sse.Data = strings.Replace(msgHTML, "\n", "", -1)

		if msg.System {
			sse.Event = "system"
			sse.Data = msg.Message
		}
		fmt.Printf("\n sse: %+v \n", sse)

		return sse

	}

	return *broker
}
