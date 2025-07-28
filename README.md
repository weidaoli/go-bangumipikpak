
一个基于 Go 语言开发的番剧自动下载工具，通过监控 RSS 源自动下载新番剧到 PikPak 云盘，并支持 QQ 和 Telegram 通知。

## 快速开始

### 环境要求

- Go 1.24 或更高版本
- PikPak 账号
- （可选）QQ 机器人（[NapCat](https://github.com/NapNeko/NapCatQQ)） API 或 Telegram Bot

### 配置说明

编辑 `config.json` 文件：

```json
{
  "pikpak": {
    "user": "your_email@example.com",
    "passwd": "your_password",
    "folder_id": "",
    "folder_path": "/番剧下载"
  },
  "rss": {
    "urls": [
      "https://example.com/rss.xml",
      "https://another-rss-source.com/feed.xml"
    ],
    "check_interval_minutes": 5,
    "keywords": ["1080p", "简体"],
    "exclude_keywords": ["720p", "繁体"],
    "resolutions": ["1080p", "2160p"]
  },
  "qq": {
    "enabled": true,
    "bot_url": "http://your-qq-bot-api.com/send_private_msg",
    "token": "your_qq_bot_token",
    "notify_users": ["123456789", "987654321"]
  },
  "telegram": {
    "enabled": true,
    "token": "your_telegram_bot_token",
    "chat_id": -1001234567890
  }
}
```

### 运行
克隆项目
```bash
git clone https://github.com/weidaoli/go-bangumipikpak.git
```

编译运行：

```bash
go build -o bangumipikpak
./bangumipikpak
```
docker run运行：
```bash
docker build -t bangumipikpak:latest .
docker run -d \
  --name bangumipikpak \
  --restart unless-stopped \
  -v ./config.json:/app/config.json:ro \
  bangumipikpak:latest
```
使用 Docker Compose
```bash
#准备配置文件
vim config.json
#运行
docker compose up -d
```

##  配置详解

### PikPak 配置

| 字段 | 说明 | 必填 |
|------|------|------|
| `user` | PikPak 账号邮箱 | ✅ |
| `passwd` | PikPak 账号密码 | ✅ |
| `folder_id` | 目标文件夹 ID（可选） | ❌ |
| `folder_path` | 目标文件夹路径 | ❌ |

### RSS 配置

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `urls` | RSS 源地址列表 | `[]` |
| `check_interval_minutes` | 检查间隔（分钟） | `5` |
| `keywords` | 包含关键词过滤 | `[]` |
| `exclude_keywords` | 排除关键词过滤 | `[]` |
| `resolutions` | 分辨率过滤 | `[]` |

### QQ 通知配置

| 字段 | 说明 | 必填 |
|------|------|------|
| `enabled` | 是否启用 QQ 通知 | ❌ |
| `bot_url` | QQ 机器人 API 地址 | ❌ |
| `token` | QQ 机器人认证令牌 | ❌ |
| `notify_users` | 通知用户 QQ 号列表 | ❌ |

### Telegram 通知配置

| 字段 | 说明 | 必填 |
|------|------|------|
| `enabled` | 是否启用 Telegram 通知 | ❌ |
| `token` | Telegram Bot Token | ❌ |
| `chat_id` | 聊天 ID（个人或群组） | ❌ |

## 高级功能

### 文件名清理

程序会自动清理文件名：
- 移除 HTML 标签
- 移除方括号内容（字幕组信息）
- 替换非法文件名字符
- 限制文件名长度（200字符）

### 智能过滤

1. **关键词过滤**：只下载包含指定关键词的番剧
2. **排除关键词**：跳过包含排除关键词的番剧
3. **分辨率过滤**：只下载指定分辨率的番剧
4. **时间过滤**：只处理最近发布的内容，避免首次运行下载历史内容

### 通知功能

支持两种通知方式：
- **QQ 通知**：通过 QQ 机器人 API 发送私聊消息
- **Telegram 通知**：通过 Telegram Bot 发送消息

通知内容包括：
- 番剧标题
- 清理后的文件名
- 下载时间

##  项目结构

```
bangumipikpak/
├── main.go          # 主程序入口和 RSS 监控逻辑
├── pikpak.go        # PikPak 云盘集成
├── qq.go            # QQ 机器人通知
├── telegram.go      # Telegram 通知
├── config.json      # 配置文件
├── go.mod           # Go 模块文件
├── go.sum           # 依赖校验文件
└── README.md        # 项目说明
```

##  依赖项目

- [go-resty/resty](https://github.com/go-resty/resty) - HTTP 客户端
- [lyqingye/pikpak-go](https://github.com/lyqingye/pikpak-go) - PikPak API 客户端
- [go-telegram-bot-api/telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) - Telegram Bot API

##  注意事项

1. **账号安全**：请妥善保管 PikPak 账号密码，建议使用专门的下载账号
2. **RSS 源**：确保 RSS 源稳定可访问
3. **网络环境**：程序需要稳定的网络环境访问 RSS 源和 PikPak API
4. **存储空间**：注意 PikPak 账号的存储空间限制
5. **合法使用**：请确保下载的内容符合当地法律法规


## 常见问题

1. **PikPak 登录失败**
   - 检查用户名和密码是否正确
   - 确认网络连接正常
   - 检查是否需要验证码

2. **RSS 获取失败**
   - 检查 RSS 源地址是否正确
   - 确认 RSS 源是否可访问
   - 检查网络防火墙设置

3. **通知发送失败**
   - 检查 QQ/Telegram 配置是否正确
   - 确认 Bot Token 是否有效
   - 检查用户 ID 或聊天 ID 是否正确


##  致谢

- 感谢 [lyqingye/pikpak-go](https://github.com/lyqingye/pikpak-go) 提供的 PikPak API 客户端
- 感谢 [Bangumi-PikPak](https://github.com/YinBuLiao/Bangumi-PikPak)提供的思路参考

---

如果这个项目对你有帮助，请给个 ⭐ Star！


