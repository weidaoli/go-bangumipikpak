package main

type Config struct {
	Pikpak struct {
		Passwd     string `json:"passwd"`
		User       string `json:"user"`
		FolderID   string `json:"folder_id"`
		FolderPath string `json:"folder_path"`
	} `json:"pikpak"`
	RSS struct {
		URLs                 []string `json:"urls"`
		CheckIntervalMinutes int      `json:"check_interval_minutes"`
		Keywords             []string `json:"keywords"`
		ExcludeKeywords      []string `json:"exclude_keywords"`
		Resolutions          []string `json:"resolutions"`
	} `json:"rss"`
	QQ struct {
		Enabled     bool     `json:"enabled"`
		BotURL      string   `json:"bot_url"`
		Token       string   `json:"token"`
		NotifyUsers []string `json:"notify_users"`
	} `json:"qq"`
	Telegram struct {
		Enabled bool   `json:"enabled"`
		Token   string `json:"token"`
		ChatID  int64  `json:"chat_id"`
	} `json:"telegram"`
}
