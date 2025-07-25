package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// QQBot 表示与 QQ 机器人 API 交互的客户端
type QQBot struct {
	URL   string // API 的 URL
	Token string // 认证令牌
}

// NewQQBot 创建一个新的 QQ 机器人客户端，使用指定的 URL 和令牌
func NewQQBot(url, token string) *QQBot {
	return &QQBot{
		URL:   url,
		Token: token,
	}
}

// SendPrivateMessage 向特定用户发送私信
func (bot *QQBot) SendPrivateMessage(userID, message string) (string, error) {
	// 使用结构体和 json.Marshal 来确保生成有效的 JSON
	type TextData struct {
		Text string `json:"text"`
	}

	type MessageItem struct {
		Type string   `json:"type"`
		Data TextData `json:"data"`
	}

	type RequestPayload struct {
		UserID  string        `json:"user_id"`
		Message []MessageItem `json:"message"`
	}

	payload := RequestPayload{
		UserID: userID,
		Message: []MessageItem{
			{
				Type: "text",
				Data: TextData{
					Text: message,
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("JSON编码请求失败: %w", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", bot.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", bot.Token)

	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	return string(body), nil
}
