package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	irc "github.com/fluffle/goirc/client"
	"github.com/pteichman/fate"
)

// Backoff policy, milliseconds per attempt. End up with 30s attempts.
var backoff = []int{0, 0, 10, 30, 100, 300, 1000, 3000, 10000, 30000}

func backoffDuration(i int) time.Duration {
	if i < len(backoff) {
		return time.Duration(backoff[i]) * time.Millisecond
	}

	return time.Duration(backoff[len(backoff)-1]) * time.Millisecond
}

func backoffConnect(conn *irc.Conn, c Config) {
	for i := 0; true; i++ {
		wait := backoffDuration(i)
		time.Sleep(wait)

		err := conn.Connect()
		if err == nil {
			// The connection was successful.
			break
		}

		log.Printf("Connection to %s failed: %s [%dms]", c.Server, err,
			int64(wait/time.Millisecond))
	}
}

func RunForever(m *fate.Model, c Config) {
	stop := make(chan bool)
	conn := irc.SimpleClient(c.Nick)

	config := conn.Config()
	config.Server = c.Server
	config.Pass = c.Password

	if c.SSL {
		config.SSL = true
		config.SSLConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	me := conn.Me()
	me.Ident = c.Nick
	me.Name = c.Nick

	conn.HandleFunc("connected", func(conn *irc.Conn, line *irc.Line) {
		log.Printf("Connected to %s. Joining %v.", c.Server, c.Channels)
		for _, channel := range c.Channels {
			conn.Join(channel.Name, channel.Password)
		}
	})

	conn.HandleFunc("disconnected", func(conn *irc.Conn, line *irc.Line) {
		log.Printf("Disconnected from %s.", c.Server)
		backoffConnect(conn, c)
	})

	var channels []string
	for _, ch := range c.Channels {
		channels = append(channels, ch.Name)
	}

	conn.HandleFunc("privmsg", func(conn *irc.Conn, line *irc.Line) {
		user := line.Nick
		if in(c.IgnoreNicks, user) {
			log.Printf("Ignoring privmsg from %s", user)
			return
		}

		target := line.Args[0]
		if !in(channels, target) {
			log.Printf("Ignoring privmsg on %s", target)
			return
		}

		var (
			to  = ""
			msg = strings.TrimSpace(line.Args[1])
		)

		first := firstword(msg)
		if strings.HasPrefix(first, "@") {
			to = first[1:]
			msg = msg[len(first):]
		} else if strings.HasSuffix(first, ":") {
			to = first[:len(first)-1]
			msg = msg[len(first):]
		}

		msg = strings.TrimSpace(msg)

		log.Printf("Learn: %s", msg)
		m.Learn(msg)

		if to == c.Nick {
			go func() {
				delay := time.After(250 * time.Millisecond)
				reply := fate.QuoteFix(m.Reply(msg))
				<-delay

				log.Printf("Reply: %s", reply)
				conn.Privmsg(target, fmt.Sprintf("%s: %s", user, reply))
			}()
		}
	})

	backoffConnect(conn, c)
	<-stop
}

func firstword(s string) string {
	i := strings.Index(s, " ")
	if i == -1 {
		return s
	}
	return s[:i]
}

func in(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle || strings.Index(h, needle+":") == 0 {
			return true
		}
	}

	return false
}
