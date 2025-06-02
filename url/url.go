package url

import (
	"crypto/tls"
	"io"
	"net"
	"strconv"
	"strings"
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

func (u *URL) Request() string {
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

	request := "GET " + u.path + " HTTP/1.0\r\n"
	request += "Host: " + u.host + "\r\n"
	request += "\r\n"
	encoded := []byte(request)
	_, err = conn.Write(encoded)
	if err != nil {
		panic("Failed to send request: " + err.Error())
	}

	buf := make([]byte, 4096)
	var response strings.Builder
	for {
		read, err := conn.Read(buf)

		if read > 0 {
			response.Write(buf[:read])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			panic("Failed to read response: " + err.Error())
		}
		if read == 0 {
			break // No more data to read
		}
	}
	lines := strings.SplitAfter(response.String(), "\r\n")
	// statusLine := strings.SplitN(lines[0], " ", 3)
	// version, status, explanation := statusLine[0], statusLine[1], statusLine[2]
	responseHeaders := make(map[string]string)
	var lineNum int
	for i, line := range lines[1:] {
		if line == "\r\n" {
			lineNum = i + 1 + 1 // +1 for the status line, +1 for the empty line after headers
			break // End of headers
		}
		headerParts := strings.SplitN(line, ":", 2)
		responseHeaders[strings.ToLower(headerParts[0])] = strings.TrimSpace(headerParts[1])
	}

	if _, ok := responseHeaders["transfer-encoding"]; ok {
		panic("Transfer-Encoding header found in response, unsupported")
	}
	if _, ok := responseHeaders["content-encoding"]; ok {
		panic("Content-Encoding header found in response, unsupported")
	}

	content := strings.Join(lines[lineNum:], "")
	return content
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