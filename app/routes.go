// ================================================================================
// All HTTP routes are defined here, purely for code organisation purposes.
// ================================================================================

package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/benc-uk/go-rest-api/pkg/sse"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

var allowed = [2]string{"K", "S"}

// Simply adds all the routes to the Echo router
func addRoutes(e *echo.Echo, broker sse.Broker[ChatMessage], db *sql.DB) {
	//
	// Root route renders the main index.html template
	//
	e.GET("/", func(c echo.Context) error {
		sess, _ := session.Get("session", c)

		return c.Render(http.StatusOK, "index", map[string]any{
			// Username might be empty or nil, the template will handle it
			"username": sess.Values["username"],
		})
	})

	//
	// Login POST will set the username in the session and render the chat view
	//
	e.POST("/login", func(c echo.Context) error {
		username := c.FormValue("username")
		if username == "" {
			return c.Render(http.StatusOK, "login", map[string]any{
				"error": "Username can not be empty.",
			})
		}
		var found = false
		for _, user := range allowed {
			if user == username {
				found = true
			}
		}
		if found == false {
			return c.Render(http.StatusOK, "login", map[string]any{
				"error": "wrong",
			})
		}

		// Check if name exists
		activeUsers := broker.GetClients()
		for _, user := range activeUsers {
			if user == username {
				return c.Render(http.StatusOK, "login", map[string]any{
					"error": "That name is already taken, please pick another name.",
				})
			}
		}

		sess, _ := session.Get("session", c)
		sess.Values["username"] = username
		err := sess.Save(c.Request(), c.Response())
		if err != nil {
			log.Println("Session error: ", err)
			return c.Render(http.StatusOK, "login", map[string]any{
				"error": err.Error(),
			})
		}

		// Got this far, we can render the chat template
		return c.Render(http.StatusOK, "chat", map[string]any{
			"username":       username,
			"addLoginButton": true,
		})
	})

	//
	// Connect clients to the chat stream using the broker
	//
	e.GET("/chat-stream", func(c echo.Context) error {
		sess, _ := session.Get("session", c)
		fmt.Printf("\n sess q %s \n", c.QueryParam("q"))
		if sess.Values["username"] == nil {
			return c.Render(http.StatusOK, "login", nil)
		}
		username := sess.Values["username"].(string)

		activeUsers := broker.GetClients()
		for _, user := range activeUsers {
			if user == username {
				return c.Render(http.StatusOK, "login", nil)
				// return c.Render(http.StatusOK, "login", map[string]any{
				// 	"error": "open on another tab",
				// })
			}
		}

		return broker.Stream(username, c.Response().Writer, *c.Request())
	})

	//
	// Post messages to the chat for broadcast
	//
	e.POST("/chat", func(c echo.Context) error {
		msgText := c.FormValue("message")
		username := c.FormValue("username")
		loc, _ := time.LoadLocation("Asia/Kolkata")

		// Trim the message and handle newlines
		msgText = strings.TrimSpace(msgText)
		msgText = strings.Replace(msgText, "\n", "<br>", -1)

		if msgText == "" {
			return c.HTML(http.StatusBadRequest, "")
		}

		msg := ChatMessage{
			Username:  username,
			Message:   msgText,
			Timestamp: time.Now().In(loc).Format("15:04:05"),
		}

		// Push the new chat message to broker & store
		broker.SendToGroup("*", msg)
		storeMessage(db, msg)

		return c.HTML(http.StatusOK, "")
	})

	//
	// Used to logout
	//
	e.POST("/logout", func(c echo.Context) error {
		sess, _ := session.Get("session", c)
		sess.Values["username"] = ""

		err := sess.Save(c.Request(), c.Response())
		if err != nil {
			log.Println("Session error: ", err)
		}

		return c.Render(http.StatusOK, "login", nil)
	})

	//
	// Display the 'about' modal popup
	//
	e.GET("/modal-about", func(c echo.Context) error {
		ver := os.Getenv("VERSION")
		if ver == "" {
			ver = "Unknown!"
		}

		return c.Render(http.StatusOK, "modal-about", map[string]any{
			"version": ver,
		})
	})

	//
	// Display the users list in a modal popup
	//
	e.GET("/modal-users", func(c echo.Context) error {
		users := broker.GetClients()

		return c.Render(http.StatusOK, "modal-users", map[string]any{
			"users": users,
		})
	})

	//
	// Display the users list
	//
	e.GET("/users", func(c echo.Context) error {
		users := broker.GetClients()

		// Return users as a basic HTML list
		var html string
		for _, user := range users {
			html += "<li>" + user + "</li>"
		}

		return c.HTML(http.StatusOK, html)
	})
}
