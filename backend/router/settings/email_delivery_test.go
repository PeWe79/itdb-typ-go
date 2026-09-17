package settings

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"testing"
)

func TestValidateSMTPAddress(t *testing.T) {
	valid := []string{"user@example.com", "Display Name <user@example.com>"}
	for _, value := range valid {
		if err := validateSMTPAddress("收件人", value); err != nil {
			t.Errorf("validateSMTPAddress(收件人, %q) error = %v", value, err)
		}
	}
	invalid := map[string]string{
		"":                                 "收件人地址格式不正确",
		"not-an-email":                     "收件人地址格式不正确",
		"user@example.com\r\nBcc: x@y.com": "收件人地址不能包含换行符",
		"<user@example.com>":               "收件人地址格式不正确",
		"@example.com":                     "收件人地址格式不正确",
	}
	for value, expect := range invalid {
		err := validateSMTPAddress("收件人", value)
		if err == nil {
			t.Errorf("validateSMTPAddress(收件人, %q) expected error", value)
			continue
		}
		if err.Error() != expect {
			t.Errorf("validateSMTPAddress(收件人, %q) error = %q, want %q", value, err.Error(), expect)
		}
	}
	if err := validateSMTPAddress("发件人", "not-an-email"); err == nil || !strings.HasPrefix(err.Error(), "发件人") {
		t.Errorf("sender field name should prefix the error, got %v", err)
	}
}

func TestSMTPPlainAuthRequiresExplicitOptIn(t *testing.T) {
	config := map[string]any{
		"smtpHost": "127.0.0.1",
		"smtpPort": "1",
		"username": "user",
		"password": "password",
		"from":     "user@example.com",
	}
	if err := SendSettingsTestEmail(config, "recipient@example.com"); err == nil || err.Error() != "SMTP 明文认证未启用" {
		t.Fatalf("SendSettingsTestEmail error = %v, want explicit plain-auth error", err)
	}
}

// localSMTPTestServer 本地假 SMTP 服务器，捕获 AUTH 命令与 DATA 邮件内容
type localSMTPTestServer struct {
	port     int
	messages <-chan string
	auths    <-chan string
	errs     <-chan error
}

func startLocalSMTPTestServer(t *testing.T) localSMTPTestServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	messages := make(chan string, 1)
	auths := make(chan string, 1)
	errs := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			errs <- acceptErr
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		write := func(text string) error { _, err := fmt.Fprintf(conn, "%s\r\n", text); return err }
		if err := write("220 localhost ESMTP"); err != nil {
			errs <- err
			return
		}
		var message strings.Builder
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				errs <- readErr
				return
			}
			command := strings.TrimRight(line, "\r\n")
			switch {
			case strings.HasPrefix(strings.ToUpper(command), "EHLO"), strings.HasPrefix(strings.ToUpper(command), "HELO"):
				if err := write("250-localhost\r\n250 OK"); err != nil {
					errs <- err
					return
				}
			case strings.HasPrefix(strings.ToUpper(command), "AUTH"):
				auths <- command
				if err := write("235 auth ok"); err != nil {
					errs <- err
					return
				}
			case strings.HasPrefix(strings.ToUpper(command), "MAIL FROM:"):
				if err := write("250 sender ok"); err != nil {
					errs <- err
					return
				}
			case strings.HasPrefix(strings.ToUpper(command), "RCPT TO:"):
				if err := write("250 recipient ok"); err != nil {
					errs <- err
					return
				}
			case strings.EqualFold(command, "DATA"):
				if err := write("354 end with <CRLF>.<CRLF>"); err != nil {
					errs <- err
					return
				}
				for {
					dataLine, dataErr := reader.ReadString('\n')
					if dataErr != nil {
						errs <- dataErr
						return
					}
					dataLine = strings.TrimRight(dataLine, "\r\n")
					if dataLine == "." {
						break
					}
					message.WriteString(dataLine)
					message.WriteString("\n")
				}
				messages <- message.String()
				if err := write("250 message accepted"); err != nil {
					errs <- err
					return
				}
			case strings.EqualFold(command, "QUIT"):
				_ = write("221 bye")
				return
			default:
				if err := write("250 OK"); err != nil {
					errs <- err
					return
				}
			}
		}
	}()
	return localSMTPTestServer{port: listener.Addr().(*net.TCPAddr).Port, messages: messages, auths: auths, errs: errs}
}

func TestSendSettingsTestEmailPlainInsecureAuth(t *testing.T) {
	server := startLocalSMTPTestServer(t)
	config := map[string]any{
		"smtpHost": "127.0.0.1", "smtpPort": strconv.Itoa(server.port),
		"username": "user", "password": "password",
		"from": "sender@example.com", "allowInsecureAuth": true,
	}
	if err := SendSettingsTestEmail(config, "recipient@example.com"); err != nil {
		t.Fatal(err)
	}
	expected := "AUTH PLAIN " + base64.StdEncoding.EncodeToString([]byte("\x00user\x00password"))
	select {
	case authLine := <-server.auths:
		if authLine != expected {
			t.Fatalf("AUTH line = %q, want %q", authLine, expected)
		}
	case err := <-server.errs:
		t.Fatal(err)
	}
}

func TestBuildSettingsEmailMessageUsesFromName(t *testing.T) {
	from, err := mail.ParseAddress("sender@example.com")
	if err != nil {
		t.Fatal(err)
	}
	to, err := mail.ParseAddress("recipient@example.com")
	if err != nil {
		t.Fatal(err)
	}
	message := string(buildSettingsEmailMessage(from, to, "ITDB 运维", "测试主题", "测试正文", "text/plain"))
	if !strings.Contains(message, "From: =?utf-8?") || !strings.Contains(message, "sender@example.com") {
		t.Fatalf("message does not contain encoded sender name/address: %q", message)
	}
}

func TestSendSettingsTestEmailAgainstLocalSMTPServer(t *testing.T) {
	server := startLocalSMTPTestServer(t)
	config := map[string]any{
		"smtpHost": "127.0.0.1", "smtpPort": strconv.Itoa(server.port),
		"from": "sender@example.com", "fromName": "ITDB 运维",
	}
	if err := SendSettingsTestEmail(config, "recipient@example.com"); err != nil {
		t.Fatal(err)
	}
	select {
	case message := <-server.messages:
		if !strings.Contains(message, "sender@example.com") || !strings.Contains(message, "=?utf-8?") {
			t.Fatalf("SMTP message missing sender header: %q", message)
		}
	case err := <-server.errs:
		t.Fatal(err)
	}
}

func TestSendSettingsTestEmailPasswordResetHTMLTemplate(t *testing.T) {
	server := startLocalSMTPTestServer(t)
	config := map[string]any{
		"smtpHost": "127.0.0.1", "smtpPort": strconv.Itoa(server.port),
		"from": "sender@example.com", "fromName": "ITDB 运维",
		"resetCode":      "344355",
		"resetBrand":     "ITDB",
		"resetUsername":  "alice",
		"resetExpiresAt": "2026-09-08 21:32:26",
		"resetRequestIP": "192.168.1.2",
	}
	if err := SendSettingsTestEmail(config, "recipient@example.com"); err != nil {
		t.Fatal(err)
	}
	select {
	case message := <-server.messages:
		for _, expected := range []string{
			"Content-Type: text/html",
			"Subject: ITDB 密码找回验证码",
			"ITDB 密码找回",
			"alice",
			">344355<",
			"2026-09-08 21:32:26",
			"192.168.1.2",
			"请使用以下验证码完成密码重置",
		} {
			if !strings.Contains(message, expected) {
				t.Fatalf("reset email missing %q: %s", expected, message)
			}
		}
	case err := <-server.errs:
		t.Fatal(err)
	}
}
