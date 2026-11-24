package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"vk-music-downloader-go/core/config"
	"vk-music-downloader-go/core/logger"
)

var (
	ErrNoBrowserFound = errors.New("поддерживаемый браузер (Chrome, Edge, Brave, Chromium) не найден")
	ErrBrowserClosed  = errors.New("окно браузера было закрыто до завершения авторизации")
	ErrSessionExpired = errors.New("сессия браузера истекла, требуется интерактивный вход")
	ErrAuthTimeout    = errors.New("время ожидания авторизации истекло")
)

// CDPTarget represents a browser tab/page target in Chrome DevTools Protocol.
type CDPTarget struct {
	ID                   string `json:"id"`
	Title                string `json:"title"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// GetProfileDir returns the absolute path to the persistent browser profile.
func GetProfileDir() string {
	abs, err := filepath.Abs(".browser-profile")
	if err != nil {
		return ".browser-profile"
	}
	_ = os.MkdirAll(abs, 0755)
	return abs
}

// HasSavedSession checks if the browser profile contains existing session data.
func HasSavedSession() bool {
	dir := GetProfileDir()
	if _, err := os.Stat(dir); err != nil {
		return false
	}

	cookieCandidates := []string{
		filepath.Join(dir, "Default", "Network", "Cookies"),
		filepath.Join(dir, "Default", "Cookies"),
		filepath.Join(dir, "Cookies"),
		filepath.Join(dir, "cookies.sqlite"), // Firefox
	}

	for _, c := range cookieCandidates {
		if fi, err := os.Stat(c); err == nil && fi.Size() > 0 {
			return true
		}
	}

	defaultDir := filepath.Join(dir, "Default")
	if entries, err := os.ReadDir(defaultDir); err == nil && len(entries) > 3 {
		return true
	}

	return false
}

var (
	activeCmdsMu sync.Mutex
	activeCmds   = make(map[int]*exec.Cmd)
)

func registerActiveBrowser(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	activeCmdsMu.Lock()
	defer activeCmdsMu.Unlock()
	activeCmds[cmd.Process.Pid] = cmd
}

func unregisterActiveBrowser(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	activeCmdsMu.Lock()
	defer activeCmdsMu.Unlock()
	delete(activeCmds, cmd.Process.Pid)
}

// CleanupActiveBrowsers принудительно закрывает все запущенные нами процессы браузера (включая всё дерево дочерних процессов).
// Вызывается при закрытии Wails GUI приложения.
func CleanupActiveBrowsers() {
	activeCmdsMu.Lock()
	cmds := make([]*exec.Cmd, 0, len(activeCmds))
	for _, cmd := range activeCmds {
		cmds = append(cmds, cmd)
	}
	activeCmds = make(map[int]*exec.Cmd)
	activeCmdsMu.Unlock()

	for _, cmd := range cmds {
		if cmd != nil && cmd.Process != nil {
			pid := cmd.Process.Pid
			if runtime.GOOS == "windows" {
				_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
			} else {
				_ = cmd.Process.Kill()
			}
		}
	}
}

func findFreePort() int {
	for p := 9222; p < 9300; p++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", p), 80*time.Millisecond)
		if err != nil {
			return p
		}
		_ = conn.Close()
	}
	return 9222
}

// closeBrowserGracefully закрывает ТОЛЬКО конкретный процесс браузера, который запустило наше приложение.
// Никогда не закрывает чужие вкладки и не трогает другие процессы!
func closeBrowserGracefully(client *http.Client, port int, cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	unregisterActiveBrowser(cmd)
	pid := cmd.Process.Pid
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
	} else {
		_ = cmd.Process.Kill()
	}
}

type CDPVersion struct {
	Browser              string `json:"Browser"`
	ProtocolVersion      string `json:"Protocol-Version"`
	UserAgent            string `json:"User-Agent"`
	V8Version            string `json:"V8-Version"`
	WebKitVersion        string `json:"WebKit-Version"`
	WebSocketDebuggerUrl string `json:"webSocketDebuggerUrl"`
}

func checkCDP(port int) (*CDPVersion, bool) {
	client := &http.Client{Timeout: 600 * time.Millisecond}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/json/version", port))
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	var v CDPVersion
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, false
	}
	return &v, true
}

func openTabViaCDP(port int, targetURL string) (*CDPTarget, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/json/new?%s", port, url.QueryEscape(targetURL))

	req, err := http.NewRequest("PUT", endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		resp, err = client.Get(endpoint)
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()

	var target CDPTarget
	if err := json.NewDecoder(resp.Body).Decode(&target); err != nil {
		return nil, err
	}
	return &target, nil
}

func cleanupStaleProfileLocks(profileDir string) {
	if profileDir == "" {
		return
	}
	_ = os.Remove(filepath.Join(profileDir, "lockfile"))
	_ = os.Remove(filepath.Join(profileDir, "Default", "LOCK"))
}

// ParseTokenFromURL extracts access_token, user_id, and expires_in from a blank.html redirect URL.
func ParseTokenFromURL(rawURL string) (*config.Token, error) {
	parts := strings.Split(rawURL, "#")
	if len(parts) < 2 {
		return nil, fmt.Errorf("отсутствует hash-часть в URL")
	}

	params := strings.Split(parts[1], "&")
	res := make(map[string]string)
	for _, p := range params {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			res[kv[0]] = kv[1]
		}
	}

	accessToken := res["access_token"]
	if accessToken == "" {
		return nil, fmt.Errorf("access_token не найден в URL")
	}

	var userID int
	if uidStr, ok := res["user_id"]; ok {
		_, _ = fmt.Sscanf(uidStr, "%d", &userID)
	}

	var expires int64
	if expStr, ok := res["expires_in"]; ok {
		var expSec int64
		_, _ = fmt.Sscanf(expStr, "%d", &expSec)
		if expSec > 0 {
			expires = time.Now().Unix() + expSec
		} else {
			expires = math.MaxInt64
		}
	} else {
		expires = math.MaxInt64
	}

	return &config.Token{
		AccessToken: accessToken,
		UserID:      userID,
		Expires:     expires,
	}, nil
}

// cdpGetCookies fetches all cookies from a browser tab via CDP WebSocket.
func cdpGetCookies(wsURL string) ([]map[string]interface{}, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("CDP WS connect: %w", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	msg := `{"id":1,"method":"Network.getCookies","params":{"urls":["https://vk.ru","https://login.vk.ru","https://vk.com","https://login.vk.com"]}}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		return nil, fmt.Errorf("CDP WS write: %w", err)
	}

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("CDP WS read: %w", err)
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(data, &resp); err != nil {
			continue
		}
		id, _ := resp["id"].(float64)
		if id != 1 {
			continue
		}
		result, ok := resp["result"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("CDP: нет result в ответе")
		}
		cookies, ok := result["cookies"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("CDP: нет cookies в результате")
		}
		var out []map[string]interface{}
		for _, c := range cookies {
			if m, ok := c.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		return out, nil
	}
}

// fetchWebToken gets VK access token using session cookies via web_token API.
func fetchWebToken(cookieStr string) (*config.Token, error) {
	body := url.Values{}
	body.Set("app_id", "6287487")
	body.Set("version", "1")

	req, err := http.NewRequest(http.MethodPost, "https://login.vk.ru/?act=web_token", bytes.NewBufferString(body.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookieStr)
	req.Header.Set("Referer", "https://vk.ru/")
	req.Header.Set("Origin", "https://vk.ru")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("web_token запрос: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("web_token парсинг: %w", err)
	}

	// Ответ: {"type":"okay","data":{"access_token":"...","user_id":...,"expires":...}}
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		// Попробуем плоский формат
		data = result
	}

	accessToken, _ := data["access_token"].(string)
	if accessToken == "" {
		return nil, fmt.Errorf("web_token: access_token не получен (ответ: %v)", result)
	}

	var userID int
	switch v := data["user_id"].(type) {
	case float64:
		userID = int(v)
	case string:
		_, _ = fmt.Sscanf(v, "%d", &userID)
	}

	var expires int64 = math.MaxInt64
	if expRaw, ok := data["expires"]; ok {
		switch v := expRaw.(type) {
		case float64:
			expires = int64(v)
		case int64:
			expires = v
		}
	} else if expRaw, ok := data["expires_in"]; ok {
		if expFloat, ok := expRaw.(float64); ok && expFloat > 0 {
			expires = time.Now().Unix() + int64(expFloat)
		}
	}

	return &config.Token{
		AccessToken: accessToken,
		UserID:      userID,
		Expires:     expires,
	}, nil
}

// getPageList polls CDP /json and returns open page targets.
func getPageList(client *http.Client, jsonURL string) ([]CDPTarget, bool) {
	resp, err := client.Get(jsonURL)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	var targets []CDPTarget
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, true
	}
	var pages []CDPTarget
	for _, t := range targets {
		if t.Type == "page" {
			pages = append(pages, t)
		}
	}
	return pages, true
}

// loginViaExistingCDP connects to an already running browser on port and guides auth on vk.ru.
func loginViaExistingCDP(ctx context.Context, port int) (*config.Token, error) {
	httpClient := &http.Client{Timeout: 2 * time.Second}
	jsonURL := fmt.Sprintf("http://127.0.0.1:%d/json", port)

	pages, _ := getPageList(httpClient, jsonURL)
	var vkTarget *CDPTarget
	for _, p := range pages {
		u := strings.ToLower(p.URL)
		if strings.Contains(u, "vk.ru") || strings.Contains(u, "vk.com") {
			targetCopy := p
			vkTarget = &targetCopy
			break
		}
	}

	var createdTargetID string
	if vkTarget == nil {
		newTarget, err := openTabViaCDP(port, "https://vk.ru")
		if err == nil && newTarget != nil {
			vkTarget = newTarget
			createdTargetID = newTarget.ID
		}
	}

	defer func() {
		if createdTargetID != "" {
			_, _ = httpClient.Get(fmt.Sprintf("http://127.0.0.1:%d/json/close/%s", port, createdTargetID))
		}
	}()

	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			logger.Warn("Время ожидания авторизации истекло")
			return nil, ErrAuthTimeout
		case <-ticker.C:
			currentPages, ok := getPageList(httpClient, jsonURL)
			if !ok {
				return nil, ErrBrowserClosed
			}
			for _, p := range currentPages {
				if p.WebSocketDebuggerURL == "" {
					continue
				}
				cookies, err := cdpGetCookies(p.WebSocketDebuggerURL)
				if err != nil {
					continue
				}
				hasSession := false
				var cookieParts []string
				for _, c := range cookies {
					name, _ := c["name"].(string)
					value, _ := c["value"].(string)
					if name == "remixsid" && value != "" && value != "DELETED" {
						hasSession = true
					}
					if name != "" {
						cookieParts = append(cookieParts, name+"="+value)
					}
				}
				if hasSession {
					logger.Info("Вход в аккаунт ВКонтакте обнаружен. Получаем токен...")
					cookieStr := strings.Join(cookieParts, "; ")
					token, err := fetchWebToken(cookieStr)
					if err != nil {
						logger.Error("Ошибка web_token: %v", err)
						continue
					}
					logger.Info("Токен успешно получен! User ID: %d", token.UserID)
					return token, nil
				}
			}
		}
	}
}

// refreshViaExistingCDP connects to an already running browser on port and fetches refreshed token.
func refreshViaExistingCDP(ctx context.Context, port int) (*config.Token, error) {
	httpClient := &http.Client{Timeout: 2 * time.Second}
	jsonURL := fmt.Sprintf("http://127.0.0.1:%d/json", port)

	newTarget, err := openTabViaCDP(port, "https://vk.ru")
	if err != nil || newTarget == nil {
		return nil, fmt.Errorf("не удалось открыть вкладку через CDP: %w", err)
	}

	defer func() {
		_, _ = httpClient.Get(fmt.Sprintf("http://127.0.0.1:%d/json/close/%s", port, newTarget.ID))
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, ErrSessionExpired
		case <-ticker.C:
			currentPages, ok := getPageList(httpClient, jsonURL)
			if !ok {
				return nil, ErrSessionExpired
			}
			for _, p := range currentPages {
				if p.ID == newTarget.ID && p.WebSocketDebuggerURL != "" {
					cookies, err := cdpGetCookies(p.WebSocketDebuggerURL)
					if err != nil {
						continue
					}
					hasSession := false
					var parts []string
					for _, c := range cookies {
						n, _ := c["name"].(string)
						v, _ := c["value"].(string)
						if n == "remixsid" && v != "" && v != "DELETED" {
							hasSession = true
						}
						if n != "" {
							parts = append(parts, n+"="+v)
						}
					}
					if hasSession {
						token, err := fetchWebToken(strings.Join(parts, "; "))
						if err != nil {
							return nil, err
						}
						return token, nil
					}
					u := strings.ToLower(p.URL)
					if strings.Contains(u, "/login") || strings.Contains(u, "/join") {
						return nil, ErrSessionExpired
					}
				}
			}
		}
	}
}

// LoginInteractive opens the browser at vk.ru, waits for the user to log in,
// then grabs session cookies via CDP and exchanges them for an access token.
func LoginInteractive(ctx context.Context, customBrowserPath ...string) (*config.Token, error) {
	logger.Info("Инициализация входа через браузер...")

	// Если уже есть запущенный браузер с CDP на порту 9222 (не headless) — прицепляемся к нему напрямую!
	if cdpVer, ok := checkCDP(9222); ok {
		isHeadless := strings.Contains(strings.ToLower(cdpVer.Browser+cdpVer.UserAgent), "headless")
		if !isHeadless {
			logger.Info("Обнаружен запущенный браузер с CDP на порту 9222 (%s). Прицепляемся напрямую...", cdpVer.Browser)
			return loginViaExistingCDP(ctx, 9222)
		}
	}

	var bInfo *BrowserInfo
	var err error

	if len(customBrowserPath) > 0 && customBrowserPath[0] != "" {
		bInfo, err = GetBrowserByPath(customBrowserPath[0])
		if err != nil {
			logger.Warn("Указанный браузер не найден (%s), ищем доступные", customBrowserPath[0])
			bInfo, err = FindSupportedBrowser()
		}
	} else {
		bInfo, err = FindSupportedBrowser()
	}

	if err != nil || bInfo == nil {
		logger.Error("Не найден поддерживаемый браузер в системе")
		return nil, ErrNoBrowserFound
	}
	logger.Info("Используется браузер: %s", bInfo.Name)

	port := findFreePort()
	profileDir := GetProfileDir()
	cleanupStaleProfileLocks(profileDir)
	logger.Info("Профиль: %s | Порт CDP: %d", profileDir, port)

	var args []string
	if bInfo.Type == TypeFirefox {
		args = []string{
			"-profile", profileDir,
			"--remote-debugging-port", strconv.Itoa(port),
			"-no-remote", "-new-window",
			"https://vk.ru",
		}
	} else {
		args = []string{
			fmt.Sprintf("--remote-debugging-port=%d", port),
			fmt.Sprintf("--user-data-dir=%s", profileDir),
			"--new-window",
			"--no-first-run",
			"--no-default-browser-check",
			"--disable-sync",
			"https://vk.ru",
		}
	}

	cmd := exec.Command(bInfo.Path, args...)
	if err := cmd.Start(); err != nil {
		logger.Error("Ошибка запуска браузера: %v", err)
		return nil, fmt.Errorf("ошибка запуска браузера %s: %w", bInfo.Name, err)
	}
	registerActiveBrowser(cmd)
	logger.Info("Браузер запущен (PID %d). Войдите в аккаунт ВКонтакте...", cmd.Process.Pid)

	httpClient := &http.Client{Timeout: 1 * time.Second}
	jsonURL := fmt.Sprintf("http://127.0.0.1:%d/json", port)
	defer closeBrowserGracefully(httpClient, port, cmd)

	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	connected := false
	startAttempts := 0
	tokenRequested := false

	for {
		select {
		case <-timeoutCtx.Done():
			logger.Warn("Время ожидания авторизации истекло")
			return nil, ErrAuthTimeout

		case <-ticker.C:
			pages, ok := getPageList(httpClient, jsonURL)

			if !ok {
				if connected {
					// CDP перестал отвечать — браузер закрыт
					logger.Warn("Браузер закрыт пользователем")
					return nil, ErrBrowserClosed
				}
				startAttempts++
				if startAttempts > 20 {
					return nil, fmt.Errorf("браузер не запустился (нет ответа CDP на порту %d)", port)
				}
				continue
			}

			if !connected {
				connected = true
				logger.Info("Браузер открыт. Войдите в аккаунт ВКонтакте на vk.ru...")
			}

			if len(pages) == 0 {
				logger.Warn("Все вкладки закрыты")
				return nil, ErrBrowserClosed
			}

			if tokenRequested {
				continue
			}

			// Ищем страницу VK и проверяем сессию
			for _, p := range pages {
				if p.WebSocketDebuggerURL == "" {
					continue
				}

				// Проверяем куки активных вкладок
				cookies, err := cdpGetCookies(p.WebSocketDebuggerURL)
				if err != nil {
					continue
				}

				hasSession := false
				var cookieParts []string
				for _, c := range cookies {
					name, _ := c["name"].(string)
					value, _ := c["value"].(string)
					if name == "remixsid" && value != "" && value != "DELETED" {
						hasSession = true
					}
					if name != "" {
						cookieParts = append(cookieParts, name+"="+value)
					}
				}

				if hasSession {
					logger.Info("Вход в аккаунт ВКонтакте обнаружен. Получаем токен...")
					tokenRequested = true

					cookieStr := strings.Join(cookieParts, "; ")
					token, err := fetchWebToken(cookieStr)
					if err != nil {
						logger.Error("Ошибка web_token: %v", err)
						tokenRequested = false
						break
					}

					logger.Info("Токен успешно получен! User ID: %d", token.UserID)
					return token, nil
				}
			}
		}
	}
}

// RefreshTokenHeadless refreshes token in headless mode using saved browser session.
func RefreshTokenHeadless(ctx context.Context, customBrowserPath ...string) (*config.Token, error) {
	if !HasSavedSession() {
		logger.Warn("Попытка фонового обновления без существующей сессии браузера")
		return nil, ErrSessionExpired
	}

	// Если уже есть активный CDP порт (например, 9222) — пробуем тихо обновить через него
	if _, ok := checkCDP(9222); ok {
		logger.Info("Обнаружен активный порт CDP 9222. Пробуем тихо обновить токен через существующий браузер...")
		token, err := refreshViaExistingCDP(ctx, 9222)
		if err == nil && token != nil {
			logger.Info("Токен успешно обновлён через существующий процесс браузера!")
			return token, nil
		}
		logger.Warn("Обновление через существующий CDP не удалось (%v), запускаем отдельный headless процесс...", err)
	}

	logger.Info("Запуск тихого (headless) обновления токена...")
	var bInfo *BrowserInfo
	var err error

	if len(customBrowserPath) > 0 && customBrowserPath[0] != "" {
		bInfo, err = GetBrowserByPath(customBrowserPath[0])
		if err != nil {
			bInfo, err = FindSupportedBrowser()
		}
	} else {
		bInfo, err = FindSupportedBrowser()
	}

	if err != nil || bInfo == nil {
		logger.Error("Не найден браузер для headless режима")
		return nil, ErrNoBrowserFound
	}

	port := findFreePort()
	profileDir := GetProfileDir()
	cleanupStaleProfileLocks(profileDir)

	var args []string
	if bInfo.Type == TypeFirefox {
		args = []string{
			"-headless", "-profile", profileDir,
			"--remote-debugging-port", strconv.Itoa(port),
			"-no-remote", "https://vk.ru",
		}
	} else {
		args = []string{
			"--headless=new",
			fmt.Sprintf("--remote-debugging-port=%d", port),
			fmt.Sprintf("--user-data-dir=%s", profileDir),
			"--disable-gpu",
			"--no-first-run",
			"--no-default-browser-check",
			"https://vk.ru",
		}
	}

	cmd := exec.Command(bInfo.Path, args...)
	if err := cmd.Start(); err != nil {
		logger.Error("Ошибка старта headless браузера: %v", err)
		return nil, fmt.Errorf("ошибка запуска headless браузера: %w", err)
	}
	registerActiveBrowser(cmd)

	httpClient := &http.Client{Timeout: 2 * time.Second}
	jsonURL := fmt.Sprintf("http://127.0.0.1:%d/json", port)
	defer closeBrowserGracefully(httpClient, port, cmd)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			logger.Warn("Таймаут headless обновления. Сессия вероятно истекла.")
			return nil, ErrSessionExpired

		case <-ticker.C:
			pages, ok := getPageList(httpClient, jsonURL)
			if !ok {
				continue
			}

			for _, p := range pages {
				if p.WebSocketDebuggerURL == "" {
					continue
				}
				cookies, err := cdpGetCookies(p.WebSocketDebuggerURL)
				if err != nil {
					continue
				}
				hasSession := false
				var parts []string
				for _, c := range cookies {
					n, _ := c["name"].(string)
					v, _ := c["value"].(string)
					if n == "remixsid" && v != "" && v != "DELETED" {
						hasSession = true
					}
					if n != "" {
						parts = append(parts, n+"="+v)
					}
				}
				if hasSession {
					token, err := fetchWebToken(strings.Join(parts, "; "))
					if err != nil {
						logger.Warn("Headless web_token error: %v", err)
						return nil, ErrSessionExpired
					}
					logger.Info("Headless: токен обновлён! User ID: %d", token.UserID)
					return token, nil
				}
				// Попали на логин без сессии — сессия протухла
				u := strings.ToLower(p.URL)
				if strings.Contains(u, "/login") || strings.Contains(u, "/join") {
					logger.Warn("Headless: сессия браузера истекла (редирект на логин)")
					return nil, ErrSessionExpired
				}
			}
		}
	}
}
