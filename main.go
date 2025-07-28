package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// RSS RSS结构体定义
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	GUID        string    `xml:"guid"`
	Enclosure   Enclosure `xml:"enclosure"`
	Torrent     Torrent   `xml:"torrent"`
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length string `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

type Torrent struct {
	XMLName       xml.Name `xml:"torrent"`
	Xmlns         string   `xml:"xmlns,attr"`
	Link          string   `xml:"link"`
	ContentLength string   `xml:"contentLength"`
	PubDate       string   `xml:"pubDate"`
}

// 番剧监听器
type BangumiMonitor struct {
	config           *Config
	downloader       *OfflineDownloader
	seenItems        map[string]bool
	mutex            sync.RWMutex
	lastChecked      time.Time
	telegramNotifier *TelegramNotifier
}

// 创建新的番剧监听器
func NewBangumiMonitor(config *Config, downloader *OfflineDownloader) *BangumiMonitor {
	return &BangumiMonitor{
		config:      config,
		downloader:  downloader,
		seenItems:   make(map[string]bool),
		lastChecked: time.Now(),
	}
}

// 获取RSS内容
func (bm *BangumiMonitor) fetchRSS(rssURL string) (*RSS, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", rssURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置User-Agent，避免被反爬虫
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取RSS失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("RSS请求失败，状态码: %d", resp.StatusCode)
	}

	var rss RSS
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(&rss)
	if err != nil {
		return nil, fmt.Errorf("解析RSS失败: %v", err)
	}

	return &rss, nil
}

// 从描述或链接中提取磁力链接
func (bm *BangumiMonitor) extractMagnetLink(item Item) string {
	// 优先从torrent元素中提取
	if item.Torrent.Link != "" {
		log.Printf("🔗 从torrent元素获取链接: %s", item.Torrent.Link)
		return item.Torrent.Link
	}

	// 磁力链接正则表达式
	magnetRegex := regexp.MustCompile(`magnet:\?[^"'\s<>]+`)

	// 检查描述中的磁力链接
	if matches := magnetRegex.FindStringSubmatch(item.Description); len(matches) > 0 {
		log.Printf("🔗 从描述中提取磁力链接: %s", matches[0])
		return matches[0]
	}

	// 检查链接中的磁力链接
	if matches := magnetRegex.FindStringSubmatch(item.Link); len(matches) > 0 {
		log.Printf("🔗 从链接中提取磁力链接: %s", matches[0])
		return matches[0]
	}

	// 检查enclosure
	if item.Enclosure.URL != "" && strings.HasPrefix(item.Enclosure.URL, "magnet:") {
		log.Printf("🔗 从enclosure获取磁力链接: %s", item.Enclosure.URL)
		return item.Enclosure.URL
	}

	// 如果torrent.link不是磁力链接，可能是种子文件链接，需要转换
	if item.Torrent.Link != "" && strings.HasSuffix(item.Torrent.Link, ".torrent") {
		log.Printf("🔗 发现种子文件链接: %s", item.Torrent.Link)
		// 这里可以选择下载种子文件并转换为磁力链接，或者直接使用种子文件链接
		return item.Torrent.Link
	}

	log.Printf("⚠️  未找到磁力链接或种子文件: %s", item.Title)
	return ""
}

// 清理文件名
func (bm *BangumiMonitor) cleanFileName(title string) string {
	// 移除HTML标签
	htmlRegex := regexp.MustCompile(`<[^>]*>`)
	cleaned := htmlRegex.ReplaceAllString(title, "")

	// 移除方括号内容（通常是字幕组信息）
	bracketRegex := regexp.MustCompile(`\[[^\]]*\]`)
	cleaned = bracketRegex.ReplaceAllString(cleaned, "")

	// 移除圆括号内容
	// parenRegex := regexp.MustCompile(`\([^)]*\)`)
	// cleaned = parenRegex.ReplaceAllString(cleaned, "")

	// 移除不合法的文件名字符
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*]`)
	cleaned = invalidChars.ReplaceAllString(cleaned, "_")

	// 移除多余的空格和特殊字符
	cleaned = strings.TrimSpace(cleaned)
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")

	// 移除开头和结尾的下划线和空格
	cleaned = strings.Trim(cleaned, "_ ")

	// 限制文件名长度
	if len(cleaned) > 200 {
		cleaned = cleaned[:200]
	}

	// 如果清理后为空，使用时间戳
	if cleaned == "" {
		cleaned = fmt.Sprintf("番剧_%s", time.Now().Format("20060102_150405"))
	}

	return cleaned
}

// 检查是否应该下载该项目
func (bm *BangumiMonitor) shouldDownload(item Item) bool {
	title := strings.ToLower(item.Title)

	// 检查关键词过滤
	if len(bm.config.RSS.Keywords) > 0 {
		hasKeyword := false
		for _, keyword := range bm.config.RSS.Keywords {
			if strings.Contains(title, strings.ToLower(keyword)) {
				hasKeyword = true
				break
			}
		}
		if !hasKeyword {
			log.Printf("🔍 跳过（无匹配关键词）: %s", item.Title)
			return false
		}
	}

	// 检查排除关键词
	if len(bm.config.RSS.ExcludeKeywords) > 0 {
		for _, keyword := range bm.config.RSS.ExcludeKeywords {
			if strings.Contains(title, strings.ToLower(keyword)) {
				log.Printf("🚫 跳过（匹配排除关键词 '%s'）: %s", keyword, item.Title)
				return false
			}
		}
	}

	// 检查分辨率过滤
	if len(bm.config.RSS.Resolutions) > 0 {
		hasResolution := false
		for _, resolution := range bm.config.RSS.Resolutions {
			if strings.Contains(title, strings.ToLower(resolution)) {
				hasResolution = true
				break
			}
		}
		if !hasResolution {
			log.Printf("📺 跳过（无匹配分辨率）: %s", item.Title)
			return false
		}
	}

	return true
}

// 检查单个RSS源的新项目
func (bm *BangumiMonitor) checkRSSSource(rssURL string) error {
	log.Printf("🔍 检查RSS源: %s", rssURL)

	rss, err := bm.fetchRSS(rssURL)
	if err != nil {
		return fmt.Errorf("获取RSS失败: %v", err)
	}

	log.Printf("📡 获取到 %d 个RSS项目 (频道: %s)", len(rss.Channel.Items), rss.Channel.Title)

	newItemsCount := 0
	for i, item := range rss.Channel.Items {
		log.Printf("📄 处理项目 %d/%d: %s", i+1, len(rss.Channel.Items), item.Title)

		bm.mutex.Lock()
		alreadySeen := bm.seenItems[item.GUID]
		bm.mutex.Unlock()

		if !alreadySeen {
			bm.mutex.Lock()
			bm.seenItems[item.GUID] = true
			bm.mutex.Unlock()

			// 解析发布时间
			pubTime, err := bm.parsePublishTime(item.PubDate)
			if err != nil {
				log.Printf("⚠️  解析时间失败: %v, 使用当前时间", err)
				pubTime = time.Now()
			}

			// 只处理最近的项目（避免首次运行下载所有历史内容）
			if pubTime.After(bm.lastChecked) {
				log.Printf("🆕 发现新项目: %s", item.Title)
				log.Printf("   📅 发布时间: %s", pubTime.Format("2006-01-02 15:04:05"))

				if bm.shouldDownload(item) {
					magnetLink := bm.extractMagnetLink(item)
					if magnetLink != "" {
						log.Printf("🎬 准备下载: %s", item.Title)

						fileName := bm.cleanFileName(item.Title)
						log.Printf("📁 清理后文件名: %s", fileName)

						// 添加到PikPak下载
						err := bm.downloader.AddMagnetTask(fileName, magnetLink)
						if err != nil {
							log.Printf("❌ 添加下载任务失败: %v", err)
						} else {
							log.Printf("✅ 成功添加下载任务: %s", fileName)
							newItemsCount++

							// 发送通知
							bm.sendNotification(fileName, item.Title)
						}
					} else {
						log.Printf("⚠️  未找到磁力链接或种子文件: %s", item.Title)
					}
				}
			} else {
				log.Printf("⏰ 跳过旧项目: %s (发布时间: %s)", item.Title, pubTime.Format("2006-01-02 15:04:05"))
			}
		} else {
			log.Printf("👁️  跳过已见项目: %s", item.Title)
		}
	}

	if newItemsCount > 0 {
		log.Printf("📥 从 %s 添加了 %d 个新的下载任务", rssURL, newItemsCount)
	} else {
		log.Printf("📭 没有新的下载任务")
	}

	return nil
}

// 解析发布时间
func (bm *BangumiMonitor) parsePublishTime(pubDate string) (time.Time, error) {
	// 尝试多种时间格式
	formats := []string{
		time.RFC1123Z,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, pubDate); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析时间格式: %s", pubDate)
}

// 发送通知
func (bm *BangumiMonitor) sendNotification(fileName, originalTitle string) {
	// 如果启用了QQ通知
	if bm.config.QQ.Enabled && bm.config.QQ.BotURL != "" {
		message := fmt.Sprintf("🎬 新番剧下载通知\n\n📺 标题: %s\n📁 文件名: %s\n⏰ 时间: %s",
			originalTitle, fileName, time.Now().Format("2006-01-02 15:04:05"))

		// 创建QQ机器人客户端并发送消息
		bot := NewQQBot(bm.config.QQ.BotURL, bm.config.QQ.Token)

		// 向所有配置的用户发送通知
		for _, userID := range bm.config.QQ.NotifyUsers {
			response, err := bot.SendPrivateMessage(userID, message)
			if err != nil {
				log.Printf("❌ 发送QQ通知失败 (用户: %s): %v", userID, err)
			} else {
				log.Printf("✅ QQ通知发送成功 (用户: %s): %s", userID, fileName)
				log.Printf("📱 响应: %s", response)
			}
		}
	}

	// 如果启用了Telegram通知
	if bm.config.Telegram.Enabled && bm.telegramNotifier != nil {
		message := fmt.Sprintf("🎬 *新番剧下载通知*\n\n📺 *标题:* %s\n📁 *文件名:* %s\n⏰ *时间:* %s",
			originalTitle, fileName, time.Now().Format("2006-01-02 15:04:05"))

		err := bm.telegramNotifier.SendMessage(message)
		if err != nil {
			log.Printf("❌ 发送Telegram通知失败: %v", err)
		} else {
			log.Printf("✅ Telegram通知发送成功: %s", fileName)
		}
	}
}

// 初始化已见项目（避免首次运行下载所有历史内容）
func (bm *BangumiMonitor) initializeSeenItems() {
	log.Println("🔄 初始化已见项目...")

	totalItems := 0
	for i, rssURL := range bm.config.RSS.URLs {
		log.Printf("📡 初始化RSS源 %d/%d: %s", i+1, len(bm.config.RSS.URLs), rssURL)

		rss, err := bm.fetchRSS(rssURL)
		if err != nil {
			log.Printf("❌ 初始化RSS源失败: %v", err)
			continue
		}

		bm.mutex.Lock()
		for _, item := range rss.Channel.Items {
			bm.seenItems[item.GUID] = true
		}
		bm.mutex.Unlock()

		log.Printf("✅ 已标记 %d 个现有项目", len(rss.Channel.Items))
		totalItems += len(rss.Channel.Items)
	}

	log.Printf("🎯 初始化完成，共标记 %d 个现有项目", totalItems)
}

// 显示配置信息
func (bm *BangumiMonitor) showConfig() {
	log.Printf("⚙️  配置信息:")
	log.Printf("   📡 RSS源数量: %d", len(bm.config.RSS.URLs))

	for i, url := range bm.config.RSS.URLs {
		log.Printf("      %d. %s", i+1, url)
	}

	checkInterval := time.Duration(bm.config.RSS.CheckIntervalMinutes) * time.Minute
	if checkInterval == 0 {
		checkInterval = 5 * time.Minute
	}
	log.Printf("   ⏱️  检查间隔: %v", checkInterval)

	if len(bm.config.RSS.Keywords) > 0 {
		log.Printf("   🔍 关键词过滤: %v", bm.config.RSS.Keywords)
	}

	if len(bm.config.RSS.ExcludeKeywords) > 0 {
		log.Printf("   🚫 排除关键词: %v", bm.config.RSS.ExcludeKeywords)
	}

	if len(bm.config.RSS.Resolutions) > 0 {
		log.Printf("   📺 分辨率过滤: %v", bm.config.RSS.Resolutions)
	}

	log.Printf("   📱 QQ通知: %v", bm.config.QQ.Enabled)
	log.Printf("   📱 Telegram通知: %v", bm.config.Telegram.Enabled)
}

// 开始监听所有RSS源
func (bm *BangumiMonitor) StartMonitoring() {
	log.Println("🚀 启动番剧监听器...")

	// 显示配置信息
	bm.showConfig()

	// 初始化已见项目
	bm.initializeSeenItems()

	checkInterval := time.Duration(bm.config.RSS.CheckIntervalMinutes) * time.Minute
	if checkInterval == 0 {
		checkInterval = 5 * time.Minute
	}

	// 定期检查
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	log.Println("🎬 开始监听番剧更新...")
}
