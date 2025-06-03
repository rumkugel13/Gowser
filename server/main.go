package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	urllib "net/url"
	"slices"
	"strconv"
	"strings"
)

var (
	ENTRIES = []string{"First entry"}
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

	status, body := do_request(method, url, headers, body)

	response := "HTTP/1.0 " + status + "\r\n"
	response += "Content-length: " + strconv.Itoa(len(body)) + "\r\n"
	response += "\r\n" + body
	conn.Write([]byte(response))
	// closed by defer
}

func do_request(method, url string, headers map[string]string, body string) (string, string) {
	if method == "GET" && url == "/" {
		return "200 OK", show_comments()
	} else if method == "POST" && url == "/add" {
		params := form_decode(body)
		return "200 OK", add_entry(params)
	} else {
		return "404 Not Found", not_found(url, method)
	}
}

func show_comments() string {
	out := "<!doctype html>"

	out += "<form action=add method=post>"
	out += "<p><input name=guest></p>"
	out += "<p><button>Sign the book!</button></p>"
	out += "</form>"

	for _, entry := range ENTRIES {
		out += "<p>" + entry + "</p>"
	}
	return out
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

func add_entry(params map[string]string) string {
	if param, ok := params["guest"]; ok {
		ENTRIES = append(ENTRIES, param)
	}
	return show_comments()
}

func not_found(url, method string) string {
	out := "<!doctype html>"
    out += fmt.Sprintf("<h1>%s %s not found!</h1>", method, url)
    return out
}
