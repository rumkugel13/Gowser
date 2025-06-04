package url

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"strconv"
	"strings"
)

type Param struct {
	cookie string
	params map[string]string
}

var (
	COOKIE_JAR = map[string]Param{}
)

type URL struct {
	scheme string
	host   string
	path   string
	port   int
}

func NewURL(url string) *URL {
	u := &URL{}
	splitURL := strings.Split(url, "://")
	u.scheme, url = splitURL[0], splitURL[1]
	if u.scheme != "http" && u.scheme != "https" {
		panic("Unsupported URL scheme: " + u.scheme)
	}
	if u.scheme == "http" {
		u.port = 80
	} else if u.scheme == "https" {
		u.port = 443
	}
	if !strings.Contains(url, "/") {
		url += "/"
	}
	splitPath := strings.SplitN(url, "/", 2)
	u.host, url = splitPath[0], splitPath[1]
	if strings.Contains(u.host, ":") {
		hostParts := strings.Split(u.host, ":")
		u.host = hostParts[0]
		port, err := strconv.Atoi(hostParts[1])
		if err != nil {
			panic("Invalid port in URL: " + hostParts[1])
		}
		u.port = port
	}
	u.path = "/" + url
	return u
}

func (u *URL) Request(referrer *URL, payload string) (map[string]string, string) {
	// Create connection
	conn, err := net.Dial("tcp", u.host+":"+strconv.Itoa(u.port))
	if err != nil {
		panic("Failed to connect to host: " + err.Error())
	}
	defer conn.Close()
	if u.scheme == "https" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // For simplicity, skip TLS verification
		}
		conn = tls.Client(conn, tlsConfig)
		if err := conn.(*tls.Conn).Handshake(); err != nil {
			panic("Failed to perform TLS handshake: " + err.Error())
		}
	}

	// Create Request Header
	method := "GET"
	if payload != "" {
		method = "POST"
	}

	request := method + " " + u.path + " HTTP/1.0\r\n"
	if cookie, ok := COOKIE_JAR[u.host]; ok {
		cookie, params := cookie.cookie, cookie.params
		allow_cookie := true
		if referrer != nil && GetOrDefault(params, "samesite", "none") == "lax" {
			if method != "GET" {
				allow_cookie = u.host == referrer.host
			}
		}
		if allow_cookie {
			request += "Cookie: " + cookie + "\r\n"
		}
	}
	if payload != "" {
		length := len(payload)
		request += "Content-Length: " + strconv.Itoa(length) + "\r\n"
	}
	request += "Host: " + u.host + "\r\n"
	request += "\r\n"

	if payload != "" {
		request += payload
	}

	// Send Request Header
	encoded := []byte(request)
	_, err = conn.Write(encoded)
	if err != nil {
		panic("Failed to send request: " + err.Error())
	}

	// Read Response
	reader := bufio.NewReader(conn)
	_, err = reader.ReadString('\n')
	// statusline, err = reader.ReadString('\n')
	if err != nil {
		panic("Failed to read response: " + err.Error())
	}
	// split := strings.SplitN(statusline, " ", 3)
	// version, status, explanation := split[0], split[1], split[2]

	responseHeaders := make(map[string]string)
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
		responseHeaders[strings.ToLower(header)] = strings.TrimSpace(value)
	}

	if cookie, ok := responseHeaders["set-cookie"]; ok {
		params := map[string]string{}
		if strings.Contains(cookie, ";") {
			split := strings.SplitN(cookie, ";", 2)
			cookie = split[0]
			rest := split[1]
			for _, param := range strings.Split(rest, ";") {
				var value string
				if strings.Contains(param, "=") {
					split = strings.SplitN(param, "=", 2)
					param = split[0]
					value = split[1]
				} else {
					value = "true"
				}
				params[strings.ToLower(strings.TrimSpace(param))] = strings.ToLower(value)
			}
		}
		COOKIE_JAR[u.host] = Param{cookie, params}
	}

	if _, ok := responseHeaders["transfer-encoding"]; ok {
		panic("Transfer-Encoding header found in response, unsupported")
	}
	if _, ok := responseHeaders["content-encoding"]; ok {
		panic("Content-Encoding header found in response, unsupported")
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		panic("Failed to read response: " + err.Error())
	}
	return responseHeaders, string(content)
}

func (u *URL) String() string {
	port_part := ":" + strconv.Itoa(u.port)
	if u.scheme == "https" && u.port == 443 {
		port_part = ""
	}
	if u.scheme == "http" && u.port == 80 {
		port_part = ""
	}
	return u.scheme + "://" + u.host + port_part + u.path
}

func (u *URL) Resolve(link_url string) *URL {
	if strings.Contains(link_url, "://") {
		return NewURL(link_url)
	}
	if !strings.HasPrefix(link_url, "/") {
		if i := strings.LastIndex(u.path, "/"); i != -1 {
			dir := u.path[:i]
			for strings.HasPrefix(link_url, "../") {
				split := strings.SplitN(link_url, "/", 2)
				link_url = split[1]
				if strings.Contains(dir, "/") {
					if i := strings.LastIndex(u.path, "/"); i != -1 {
						dir = dir[:i]
					}
				}
			}
			link_url = dir + "/" + link_url
		}
	}
	if strings.HasPrefix(link_url, "//") {
		return NewURL(u.scheme + ":" + link_url)
	} else {
		return NewURL(u.scheme + "://" + u.host + ":" + strconv.Itoa(u.port) + link_url)
	}
}

func (u *URL) Origin() string {
	return u.scheme + "://" + u.host + ":" + strconv.Itoa(u.port)
}

func GetOrDefault(m map[string]string, param, def string) string {
	if val, ok := m[param]; ok {
		return val
	}
	return def
}
