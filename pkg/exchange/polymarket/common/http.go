package common

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// HTTPClientConfig HTTP 客户端配置
type HTTPClientConfig struct {
	BaseURL     string
	Timeout     time.Duration
	ProxyString string // 格式: host:port 或 host:port:user:pass 或 host:port:user:pass:socks5
	Debug       bool
	Retry       int
}

// HTTPClient HTTP 客户端
type HTTPClient struct {
	Client  *http.Client
	BaseURL string
	debug   bool
	retry   int
}

// NewHTTPClient 创建 HTTP 客户端
func NewHTTPClient(cfg HTTPClientConfig) *HTTPClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.Retry == 0 {
		cfg.Retry = 2
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	// 配置代理
	if cfg.ProxyString != "" {
		configureProxy(transport, cfg.ProxyString)
	}

	return &HTTPClient{
		Client: &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		},
		BaseURL: strings.TrimSuffix(cfg.BaseURL, "/"),
		debug:   cfg.Debug,
		retry:   cfg.Retry,
	}
}

// configureProxy 配置代理
func configureProxy(transport *http.Transport, proxyString string) {
	parts := strings.Split(proxyString, ":")
	if len(parts) < 2 {
		return
	}

	host := parts[0]
	port := parts[1]

	// 判断代理类型
	proxyType := "http"
	var username, password string

	if len(parts) >= 4 {
		username = parts[2]
		password = parts[3]
		if len(parts) >= 5 {
			proxyType = strings.ToLower(parts[4])
		}
	}

	if strings.HasPrefix(proxyType, "socks") {
		// SOCKS5 代理
		var auth *proxy.Auth
		if username != "" && password != "" {
			auth = &proxy.Auth{User: username, Password: password}
		}
		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%s", host, port), auth, proxy.Direct)
		if err == nil {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		}
	} else {
		// HTTP/HTTPS 代理
		proxyURL := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%s", host, port),
		}
		if username != "" && password != "" {
			proxyURL.User = url.UserPassword(username, password)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
}

// Get 发送 GET 请求
func (c *HTTPClient) Get(ctx context.Context, path string, params interface{}) ([]byte, error) {
	urlStr := c.BaseURL + path
	if params != nil {
		query := BuildQuery(params)
		if query != "" {
			urlStr += "?" + query
		}
	}

	var lastErr error
	for i := 0; i <= c.retry; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.Client.Do(req)
		if err != nil {
			lastErr = err
			if i < c.retry {
				time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("do request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 400 {
			// 可重试的状态码
			if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
				if i < c.retry {
					time.Sleep(time.Duration(i+1) * time.Second)
					continue
				}
			}
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}

		return body, nil
	}

	return nil, lastErr
}

// GetJSON 发送 GET 请求并解析 JSON
func (c *HTTPClient) GetJSON(ctx context.Context, path string, params interface{}, result interface{}) error {
	body, err := c.Get(ctx, path, params)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

// Post 发送 POST 请求
func (c *HTTPClient) Post(ctx context.Context, path string, data interface{}) ([]byte, error) {
	urlStr := c.BaseURL + path

	var bodyReader io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = strings.NewReader(string(jsonData))
	}

	var lastErr error
	for i := 0; i <= c.retry; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.Client.Do(req)
		if err != nil {
			lastErr = err
			if i < c.retry {
				time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("do request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 400 {
			if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
				if i < c.retry {
					time.Sleep(time.Duration(i+1) * time.Second)
					continue
				}
			}
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}

		return body, nil
	}

	return nil, lastErr
}

// PostJSON 发送 POST 请求并解析 JSON
func (c *HTTPClient) PostJSON(ctx context.Context, path string, data interface{}, result interface{}) error {
	body, err := c.Post(ctx, path, data)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

// BuildQuery 从结构体构建查询字符串
func BuildQuery(params interface{}) string {
	if params == nil {
		return ""
	}

	values := url.Values{}
	v := reflect.ValueOf(params)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 获取 url tag
		tag := fieldType.Tag.Get("url")
		if tag == "" || tag == "-" {
			continue
		}

		// 解析 tag
		parts := strings.Split(tag, ",")
		name := parts[0]
		omitempty := len(parts) > 1 && parts[1] == "omitempty"

		// 获取值
		var strVal string
		switch field.Kind() {
		case reflect.String:
			strVal = field.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Int() != 0 || !omitempty {
				strVal = strconv.FormatInt(field.Int(), 10)
			}
		case reflect.Bool:
			if field.Bool() || !omitempty {
				strVal = strconv.FormatBool(field.Bool())
			}
		case reflect.Float32, reflect.Float64:
			if field.Float() != 0 || !omitempty {
				strVal = strconv.FormatFloat(field.Float(), 'f', -1, 64)
			}
		case reflect.Ptr:
			if !field.IsNil() {
				elem := field.Elem()
				switch elem.Kind() {
				case reflect.Bool:
					strVal = strconv.FormatBool(elem.Bool())
				case reflect.Int, reflect.Int64:
					strVal = strconv.FormatInt(elem.Int(), 10)
				case reflect.String:
					strVal = elem.String()
				}
			}
		}

		if strVal != "" || !omitempty {
			if strVal != "" {
				values.Set(name, strVal)
			}
		}
	}

	return values.Encode()
}

// ProxyConfig 代理配置（解析后）
type ProxyConfig struct {
	Host      string
	Port      string
	Username  string
	Password  string
	ProxyType string // http, socks5
}

// ParseProxyString 解析代理字符串
func ParseProxyString(proxyString string) *ProxyConfig {
	if proxyString == "" {
		return nil
	}

	parts := strings.Split(proxyString, ":")
	if len(parts) < 2 {
		return nil
	}

	cfg := &ProxyConfig{
		Host:      parts[0],
		Port:      parts[1],
		ProxyType: "http",
	}

	if len(parts) >= 4 {
		cfg.Username = parts[2]
		cfg.Password = parts[3]
		if len(parts) >= 5 {
			cfg.ProxyType = strings.ToLower(parts[4])
		}
	}

	return cfg
}

// GetProxyURL 获取代理 URL（用于 HTTP/WebSocket）
func (c *ProxyConfig) GetProxyURL() *url.URL {
	if c == nil {
		return nil
	}
	proxyURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", c.Host, c.Port),
	}
	if c.Username != "" && c.Password != "" {
		proxyURL.User = url.UserPassword(c.Username, c.Password)
	}
	return proxyURL
}

// IsSocks 是否为 SOCKS 代理
func (c *ProxyConfig) IsSocks() bool {
	return c != nil && strings.HasPrefix(c.ProxyType, "socks")
}

// CreateProxyDialer 创建代理 Dialer（用于 SOCKS5 WebSocket）
func CreateProxyDialer(proxyString string) (proxy.Dialer, error) {
	cfg := ParseProxyString(proxyString)
	if cfg == nil {
		return proxy.Direct, nil
	}

	if cfg.IsSocks() {
		var auth *proxy.Auth
		if cfg.Username != "" && cfg.Password != "" {
			auth = &proxy.Auth{User: cfg.Username, Password: cfg.Password}
		}
		return proxy.SOCKS5("tcp", fmt.Sprintf("%s:%s", cfg.Host, cfg.Port), auth, proxy.Direct)
	}

	// HTTP 代理返回 nil，由调用方使用 GetProxyURL
	return nil, nil
}
