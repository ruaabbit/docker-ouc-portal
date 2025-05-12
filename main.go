package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultCheckIntervalSeconds = 600
	defaultCheckTargetHost      = "https://www.baidu.com/"
	defaultAuthMode             = "XHA"
)

var (
	authURLs = map[string]string{
		"XHA": "https://192.168.101.201:802/eportal/portal/login",
		// 可以根据 README 或实际情况添加其他模式的 URL
		// "WXRZ": "https://wxrz.ouc.edu.cn:802/eportal/portal/login", // 示例
		// "YXRZ": "https://yxrz.ouc.edu.cn:802/eportal/portal/login", // 示例
	}
)

func main() {
	username := getEnv("WLJF_USERNAME", "")
	password := getEnv("WLJF_PASSWORD", "")
	authMode := getEnv("WLJF_MODE", defaultAuthMode)
	checkIntervalSecondsStr := getEnv("CHECK_INTERVAL_SECONDS", strconv.Itoa(defaultCheckIntervalSeconds))
	checkTargetHost := getEnv("CHECK_TARGET_HOST", defaultCheckTargetHost)

	if username == "" {
		log.Fatal("错误：环境变量 WLJF_USERNAME 必须设置。")
	}
	if password == "" {
		log.Fatal("错误：环境变量 WLJF_PASSWORD 必须设置。")
	}

	checkIntervalSeconds, err := strconv.Atoi(checkIntervalSecondsStr)
	if err != nil {
		log.Printf("警告：CHECK_INTERVAL_SECONDS 无效 (%s)，将使用默认值 %d 秒。\n", checkIntervalSecondsStr, defaultCheckIntervalSeconds)
		checkIntervalSeconds = defaultCheckIntervalSeconds
	}
	if checkIntervalSeconds <= 0 {
		log.Printf("警告：CHECK_INTERVAL_SECONDS (%d) 必须大于 0，将使用默认值 %d 秒。\n", checkIntervalSeconds, defaultCheckIntervalSeconds)
		checkIntervalSeconds = defaultCheckIntervalSeconds
	}

	authURL, ok := authURLs[authMode]
	if !ok {
		log.Printf("警告：认证模式 WLJF_MODE (%s) 无效或未在程序中定义，将使用默认模式 %s (%s)。\n", authMode, defaultAuthMode, authURLs[defaultAuthMode])
		authURL = authURLs[defaultAuthMode]
		authMode = defaultAuthMode // 确保 authMode 变量也更新
	}

	log.Println("校园网自动认证服务启动...")
	log.Printf("用户名: %s", username)
	log.Printf("认证模式: %s", authMode)
	log.Printf("认证URL: %s", authURL)
	log.Printf("网络检测目标: %s", checkTargetHost)
	log.Printf("网络检测间隔: %d 秒", checkIntervalSeconds)

	// 首次立即执行检查和登录尝试
	log.Println("首次执行网络状态检测和认证...")
	checkAndLogin(username, password, authURL, checkTargetHost)

	ticker := time.NewTicker(time.Duration(checkIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		checkAndLogin(username, password, authURL, checkTargetHost)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("环境变量 %s 未设置，将使用默认值: %s", key, fallback)
	return fallback
}

func isNetworkConnected(targetHost string) bool {
	log.Printf("正在检测网络连接状态 (目标: %s)...", targetHost)
	// 增加 http:// 前缀以确保是有效的 URL
	// 同时，校园网未登录时，DNS 可能被劫持，直接访问 IP 或许更稳定，但 targetHost 通常是域名
	var checkURL string
	if !strings.HasPrefix(targetHost, "http://") && !strings.HasPrefix(targetHost, "https://") {
		checkURL = "http://" + targetHost // 默认使用 http
	} else {
		checkURL = targetHost
	}

	client := http.Client{
		Timeout: 5 * time.Second, // 设置超时，避免长时间阻塞
	}

	resp, err := client.Get(checkURL)
	if err != nil {
		log.Printf("网络检测失败 (无法访问 %s): %v", checkURL, err)
		return false
	}
	defer resp.Body.Close()

	// 任何 2xx 或 3xx 的状态码通常表示可以访问到目标服务器（即使内容可能不是预期的）
	// 对于校园网认证场景，能发出请求并收到响应（即使是被重定向到登录页）也可能意味着需要登录
	// 更精确的判断是能否访问 *公网* 资源。
	// 如果目标是公网，且能正常访问（如 200 OK），则认为网络已通。
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("网络已连接 (访问 %s 成功，状态码: %d)。", checkURL, resp.StatusCode)
		return true
	}

	// 有些校园网在未登录时访问 HTTP 网站会返回特定状态码或重定向
	// 例如，如果返回的是登录页面，状态码可能是 200，但内容不同
	// 这里简化处理：只要能通，就认为可能已登录或不需要登录。
	// 如果返回非 2xx，则明确认为网络不通或需要认证。
	log.Printf("网络检测到目标 %s 返回非预期状态码: %d。可能需要认证。", checkURL, resp.StatusCode)
	return false
}

func login(username, password, authURL string) {
	log.Println("尝试进行校园网认证...")

	params := url.Values{}
	params.Add("callback", "dr1003") // 根据 README 中的 cURL
	params.Add("login_method", "1")
	params.Add("user_account", username)
	params.Add("user_password", password)
	params.Add("wlan_user_ip", "0.0.0.0") // 通常由服务器自动确定或设为0.0.0.0
	params.Add("wlan_user_ipv6", "")
	params.Add("wlan_user_mac", "000000000000") // 虚拟 MAC
	params.Add("wlan_ac_ip", "")
	params.Add("wlan_ac_name", "")
	params.Add("jsVersion", "4.1")   // 与 cURL 保持一致
	params.Add("terminal_type", "1") // 与 cURL 保持一致
	params.Add("lang", "zh-cn")      // 与 cURL 保持一致

	fullURL := authURL + "?" + params.Encode()
	// 为安全起见，不在日志中完整打印包含密码的 URL
	log.Printf("发送认证请求至: %s (参数已编码)", authURL)

	client := http.Client{
		Timeout: 10 * time.Second, // 设置请求超时
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Printf("创建认证请求失败: %v", err)
		return
	}
	// 根据 cURL，可能需要设置一些特定的头部，但这里暂时不设置
	// req.Header.Set("User-Agent", "Mozilla/5.0...")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("认证请求发送失败: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取认证响应体失败: %v", err)
		return
	}

	responseStr := string(body)
	log.Printf("认证响应状态码: %d", resp.StatusCode)
	// log.Printf("原始认证响应内容: %s", responseStr) // 原始响应可能过长，选择性记录

	// 处理 dr1003(...) 格式的 JSONP 响应
	// 示例成功: dr1003({"result":1,"msg":"Portal协议认证成功！"});
	// 示例失败: dr1003({"result":0,"msg":"password_error"}); (假设 msg 是直接的错误文本)
	if strings.HasPrefix(responseStr, "dr1003(") && strings.Contains(responseStr, ")") {
		startIndex := strings.Index(responseStr, "(")
		// 查找最后一个 ')'，并考虑到末尾可能有分号，如 dr1003({...});
		endIndex := strings.LastIndex(responseStr, ")")

		// 确保括号配对且顺序正确
		if startIndex != -1 && endIndex != -1 && endIndex > startIndex {
			jsonStr := responseStr[startIndex+1 : endIndex]
			log.Printf("解析的 JSON 内容: %s", jsonStr)

			var resultData map[string]interface{}
			err := json.Unmarshal([]byte(jsonStr), &resultData)
			if err != nil {
				log.Printf("错误：解析认证响应 JSON 失败: %v。原始响应: %s", err, responseStr)
				// 回退到基本字符串检查 (基于用户反馈的成功消息)
				if strings.Contains(responseStr, `"result":1`) && strings.Contains(responseStr, `"msg":"Portal协议认证成功！"`) {
					log.Println("认证成功！(JSON 解析失败，但响应中包含明确成功指示)")
				} else {
					log.Println("认证可能失败，JSON 解析失败且未找到明确成功指示。")
				}
				return
			}

			// 检查 result 和 msg 字段
			resultVal, resultOk := resultData["result"].(float64) // JSON 数字通常解析为 float64
			msgVal, msgOk := resultData["msg"].(string)

			if resultOk && resultVal == 1.0 {
				if msgOk && msgVal == "Portal协议认证成功！" {
					log.Println("认证成功！")
				} else {
					log.Printf("认证结果为1，但消息非预期: '%s'。仍视为部分成功。", msgVal)
				}
				// 可以记录其他成功信息，如 olmass, tiyuan 等
				if olmass, ok := resultData["olmass"].(string); ok && olmass != "" {
					log.Printf("在线时长/流量信息 (olmass): %s", olmass)
				}
			} else if resultOk && resultVal == 0.0 { // 认证失败通常 result 为 0
				log.Println("认证失败。")
				if msgOk && msgVal != "" {
					log.Printf("错误信息: %s", msgVal) // 直接显示 msg 内容
				} else {
					log.Println("认证失败，未提供具体错误信息。")
				}
			} else {
				// result 字段存在但值非预期 (非0或1) 或 result 字段不存在
				log.Printf("认证响应中 result 字段值非预期或不存在。Result: %v, Msg: '%s'。完整 JSON: %s", resultData["result"], msgVal, jsonStr)
			}
		} else {
			// 括号不匹配或顺序错误
			log.Printf("无法从响应中正确提取 JSON 内容 (括号问题): %s", responseStr)
		}
	} else {
		// 非 JSONP 格式，或非预期的 dr1003 响应
		log.Printf("收到的认证响应非预期的 dr1003 JSONP 格式: %s", responseStr)
		// 可以添加基于关键字的通用检查作为后备
		if strings.Contains(responseStr, "success") || strings.Contains(responseStr, "成功") || strings.Contains(responseStr, "login_ok") {
			log.Println("认证请求已发送，响应中可能包含成功指示 (非标准格式)。")
		} else if strings.Contains(responseStr, "fail") || strings.Contains(responseStr, "失败") || strings.Contains(responseStr, "error") {
			log.Println("认证请求已发送，响应中可能包含失败指示 (非标准格式)。")
		} else {
			log.Println("认证响应格式未知，请检查日志中的原始响应内容。")
		}
	}
}

func checkAndLogin(username, password, authURL, checkTargetHost string) {
	if !isNetworkConnected(checkTargetHost) {
		// 如果网络不通，则执行登录
		login(username, password, authURL)
	} else {
		// 如果网络已通，则不执行任何操作
		// log.Println("网络连接正常，本次无需认证。")
	}
}
