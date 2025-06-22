package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"html"
	"io"
	"net"
	urllib "net/url"
	"os"
	"slices"
	"strconv"
	"strings"
)

type Entry struct {
	entry, user string
}

var (
	ENTRIES = []Entry{
		{"No names. We are nameless!", "cerealkiller"},
		{"HACK THE PLANET!!!", "crashoverride"},
	}
	SESSIONS = map[string]map[string]string{}
	LOGINS   = map[string]string{
		"crashoverride": "0cool",
		"cerealkiller":  "emmanuel",
	}
)

func main() {
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("Listening on :8000")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err)
			continue
		}

		go handle_connection(conn)
	}
}

func handle_connection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	reqline, _ := reader.ReadString('\n')
	fmt.Println("Request: " + strings.TrimSuffix(reqline, "\r\n"))
	split := strings.SplitN(reqline, " ", 3)
	method, url, _ := split[0], split[1], split[2]
	if !slices.Contains([]string{"GET", "POST"}, method) {
		panic("Unknown method: " + method)
	}

	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			panic("Failed to read response: " + err.Error())
		}
		if line == "\r\n" {
			break
		}
		split := strings.SplitN(line, ":", 2)
		header, value := split[0], split[1]
		headers[strings.ToLower(header)] = strings.TrimSpace(value)
	}

	var body string
	if val, ok := headers["content-length"]; ok {
		length, _ := strconv.Atoi(val)
		buf := make([]byte, length)
		_, err := io.ReadFull(reader, buf)
		if err != nil {
			panic("Failed to read response: " + err.Error())
		}
		body = string(buf)
		fmt.Println("\tBody: " + body)
	}

	var token string
	if val, ok := headers["cookie"]; ok {
		token = val[len("token="):]
	} else {
		token = rand.Text()
	}

	session, ok := SESSIONS[token]
	if !ok {
		session = make(map[string]string)
		SESSIONS[token] = session
	}
	status, body := do_request(session, method, url, headers, body)

	response := "HTTP/1.1 " + status + "\r\n"
	response += "Content-length: " + strconv.Itoa(len(body)) + "\r\n"
	if _, ok := headers["cookie"]; !ok {
		template := "Set-Cookie: token=%s; SameSite=Lax\r\n"
		response += fmt.Sprintf(template, token)
	}
	csp := "default-src http://localhost:8000"
	response += "Content-Security-Policy: " + csp + "\r\n"
	response += "Connection: close\r\n"
	response += "\r\n" + body
	conn.Write([]byte(response))
	// closed by defer
}

func do_request(session map[string]string, method, url string, headers map[string]string, body string) (string, string) {
	if method == "GET" && url == "/" {
		return "200 OK", show_comments(session)
	} else if method == "GET" && url == "/comment.js" {
		data, err := os.ReadFile("comment.js")
		if err != nil {
			fmt.Println("Error reading comment.js")
		}
		return "200 OK", string(data)
	} else if method == "GET" && url == "/eventloop.js" {
		data, err := os.ReadFile("eventloop.js")
		if err != nil {
			fmt.Println("Error reading eventloop.js")
		}
		return "200 OK", string(data)
	} else if method == "GET" && url == "/comment.css" {
		data, err := os.ReadFile("comment.css")
		if err != nil {
			fmt.Println("Error reading comment.css")
		}
		return "200 OK", string(data)
	} else if method == "GET" && url == "/login" {
		return "200 OK", login_form(session)
	} else if method == "POST" && url == "/add" {
		params := form_decode(body)
		add_entry(session, params)
		return "200 OK", show_comments(session)
	} else if method == "POST" && url == "/" {
		params := form_decode(body)
		return do_login(session, params)
	} else if method == "GET" && url == "/count" {
		return "200 OK", show_count()
	} else {
		return "404 Not Found", not_found(url, method)
	}
}

func show_comments(session map[string]string) string {
	out := "<!doctype html>"

	if user, ok := session["user"]; ok {
		nonce := rand.Text()
		session["nonce"] = nonce
		out += "<h1>Hello, " + user + "</h1>"
		out += "<form action=add method=post>"
		out += "<p><input name=guest></p>"
		out += "<input name=nonce type=hidden value=" + nonce + ">"
		out += "<p><button>Sign the book!</button></p>"
		out += "</form>"
	} else {
		out += "<a href=/login>Sign in to write in the guest book</a>"
	}

	out += "<link rel=stylesheet href=/comment.css>"
	out += "<strong></strong>"
	out += "<script src=/comment.js></script>"
	out += "<script src=https://example.com/evil.js></script>"

	for _, entry := range ENTRIES {
		out += "<p>" + html.EscapeString(entry.entry) + "\n"
		out += "<i>by " + html.EscapeString(entry.user) + "</i></p>"
	}
	return out
}

func login_form(session map[string]string) string {
	body := "<!doctype html>"
	body += "<form action=/ method=post>"
	body += "<p>Username: <input name=username></p>"
	body += "<p>Password: <input name=password type=password></p>"
	body += "<p><button>Log in</button></p>"
	body += "</form>"
	return body
}

func form_decode(body string) map[string]string {
	params := map[string]string{}
	for _, field := range strings.Split(body, "&") {
		split := strings.SplitN(field, "=", 2)
		name, value := split[0], split[1]
		name, _ = urllib.QueryUnescape(name)
		value, _ = urllib.QueryUnescape(value)
		params[name] = value
	}
	return params
}

func add_entry(session map[string]string, params map[string]string) {
	if _, ok := session["nonce"]; !ok {
		return
	}
	if _, ok := params["nonce"]; !ok {
		return
	}
	if session["nonce"] != params["nonce"] {
		return
	}
	if user, ok := session["user"]; !ok {
		return
	} else if param, ok := params["guest"]; ok && len(param) <= 100 {
		ENTRIES = append(ENTRIES, Entry{param, user})
	}
}

func do_login(session map[string]string, params map[string]string) (string, string) {
	username := params["username"]
	password := params["password"]
	if val, ok := LOGINS[username]; ok && val == password {
		session["user"] = username
		return "200 OK", show_comments(session)
	} else {
		out := "<!doctype html>"
		out += fmt.Sprintf("<h1>Invalid password for %s</h1>", username)
		return "401 Unauthorized", out
	}
}

func show_count() string {
	out := "<!doctype html>"
	out += "<div>"
	out += "  Let's count up to 99!"
	out += "</div>"
	out += "<div>Output</div>"
	out += "<script src=/eventloop.js></script>"
	return out
}

func not_found(url, method string) string {
	out := "<!doctype html>"
	out += fmt.Sprintf("<h1>%s %s not found!</h1>", method, url)
	return out
}
