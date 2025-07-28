package main

import (
	"encoding/json"
	"fmt"
	"github.com/lyqingye/pikpak-go"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

// OfflineDownloader 离线下载器结构体
type OfflineDownloader struct {
	client *pikpakgo.PikPakClient
	config *Config
}

// NewOfflineDownloader 创建新的离线下载器实例
func NewOfflineDownloader(configFile string) (*OfflineDownloader, error) {
	config, err := parseJSONFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %v", err)
	}

	client, err := pikpakgo.NewPikPakClient(config.Pikpak.User, config.Pikpak.Passwd)
	if err != nil {
		return nil, fmt.Errorf("创建PikPak客户端失败: %v", err)
	}

	// 登录
	err = client.Login()
	if err != nil {
		return nil, fmt.Errorf("PikPak登录失败: %v", err)
	}

	log.Printf("✅ PikPak登录成功: %s", config.Pikpak.User)

	downloader := &OfflineDownloader{
		client: client,
		config: config,
	}

	// 初始化目标文件夹
	err = downloader.initializeTargetFolder()
	if err != nil {
		log.Printf("⚠️  初始化目标文件夹失败: %v", err)
	}

	return downloader, nil
}

// TestConnection 测试PikPak连接
func (od *OfflineDownloader) TestConnection() error {
	if od.client == nil {
		return fmt.Errorf("客户端未初始化")
	}

	log.Printf("🧪 测试PikPak连接...")

	// 获取用户信息来测试连接
	meInfo, err := od.client.Me()
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %v", err)
	}

	log.Printf("✅ PikPak连接测试成功")
	log.Printf("   👤 用户名: %s", meInfo.Name)
	log.Printf("   📧 邮箱: %s", meInfo.Email)

	// 获取存储空间信息
	about, err := od.client.About()
	if err != nil {
		log.Printf("⚠️  获取存储信息失败: %v", err)
	} else {
		log.Printf("   💾 存储空间: %.2f GB / %.2f GB",
			float64(about.Quota.Usage)/(1024*1024*1024),
			float64(about.Quota.Limit)/(1024*1024*1024))
	}

	return nil
}

// AddMagnetTask 添加磁力链接下载任务
func (od *OfflineDownloader) AddMagnetTask(fileName, magnetLink string) error {
	if od.client == nil {
		return fmt.Errorf("客户端未初始化")
	}

	log.Printf("📥 开始添加离线下载任务: %s", fileName)

	// 判断是磁力链接还是种子文件链接
	if strings.HasPrefix(magnetLink, "magnet:") {
		log.Printf("🧲 磁力链接: %s", magnetLink)
	} else if strings.HasSuffix(magnetLink, ".torrent") {
		log.Printf("📄 种子文件链接: %s", magnetLink)
	} else {
		log.Printf("🔗 下载链接: %s", magnetLink)
	}

	// 获取目标文件夹ID
	targetFolderID := od.getTargetFolderID()
	if targetFolderID != "" {
		log.Printf("📁 目标文件夹ID: %s", targetFolderID)
	} else {
		log.Printf("📁 目标文件夹: 根目录")
	}

	// 使用SDK的OfflineDownload方法
	// PikPak支持磁力链接和种子文件链接
	newTask, err := od.client.OfflineDownload(fileName, magnetLink, targetFolderID)
	if err != nil {
		return fmt.Errorf("添加离线下载任务失败: %v", err)
	}

	if newTask != nil && newTask.Task != nil {
		log.Printf("✅ 离线下载任务添加成功")
		log.Printf("   📋 任务ID: %s", newTask.Task.ID)
		log.Printf("   📁 文件名: %s", fileName)
		log.Printf("   📊 状态: %s", newTask.Task.Phase)
		log.Printf("   📂 目标文件夹: %s", targetFolderID)
	}

	return nil
}

// GetTaskStatus 获取任务状态
func (od *OfflineDownloader) GetTaskStatus(taskId string) (*pikpakgo.Task, error) {
	if od.client == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}

	log.Printf("🔍 查询任务状态: %s", taskId)

	// 获取任务列表并查找指定任务
	taskList, err := od.client.OfflineList(100, "")
	if err != nil {
		return nil, fmt.Errorf("获取任务列表失败: %v", err)
	}

	for _, task := range taskList.Tasks {
		if task.ID == taskId {
			log.Printf("✅ 找到任务: %s", task.Name)
			log.Printf("   📊 状态: %s", task.Phase)
			log.Printf("   📈 进度: %.2f%%", task.Progress*100)
			return task, nil
		}
	}

	return nil, fmt.Errorf("未找到任务: %s", taskId)
}

// ListTasks 列出所有任务
func (od *OfflineDownloader) ListTasks() ([]*pikpakgo.Task, error) {
	if od.client == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}

	log.Printf("📋 获取任务列表...")

	var allTasks []*pikpakgo.Task

	// 使用迭代器获取所有任务
	err := od.client.OfflineListIterator(func(task *pikpakgo.Task) bool {
		allTasks = append(allTasks, task)
		log.Printf("   📄 %s - %s (%.1f%%)", task.Name, task.Phase, task.Progress*100)
		return true // 继续迭代
	})

	if err != nil {
		return nil, fmt.Errorf("获取任务列表失败: %v", err)
	}

	log.Printf("✅ 任务列表获取完成，共 %d 个任务", len(allTasks))
	return allTasks, nil
}

// RemoveTask 删除任务
func (od *OfflineDownloader) RemoveTask(taskId string, deleteFiles bool) error {
	if od.client == nil {
		return fmt.Errorf("客户端未初始化")
	}

	log.Printf("🗑️  删除任务: %s (删除文件: %v)", taskId, deleteFiles)

	err := od.client.OfflineRemove([]string{taskId}, deleteFiles)
	if err != nil {
		return fmt.Errorf("删除任务失败: %v", err)
	}

	log.Printf("✅ 任务删除成功")
	return nil
}

// RetryTask 重试任务
func (od *OfflineDownloader) RetryTask(taskId string) error {
	if od.client == nil {
		return fmt.Errorf("客户端未初始化")
	}

	log.Printf("🔄 重试任务: %s", taskId)

	err := od.client.OfflineRetry(taskId)
	if err != nil {
		return fmt.Errorf("重试任务失败: %v", err)
	}

	log.Printf("✅ 任务重试成功")
	return nil
}

// GetFileList 获取文件列表
func (od *OfflineDownloader) GetFileList(parentId string) ([]*pikpakgo.File, error) {
	if od.client == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}

	log.Printf("📁 获取文件列表...")

	files, err := od.client.FileListAll(parentId)
	if err != nil {
		return nil, fmt.Errorf("获取文件列表失败: %v", err)
	}

	log.Printf("✅ 文件列表获取完成，共 %d 个文件", len(files))
	for _, file := range files {
		log.Printf("   📄 %s (%s)", file.Name, file.Kind)
	}

	return files, nil
}

// WaitForTaskComplete 等待任务完成
func (od *OfflineDownloader) WaitForTaskComplete(taskId string, timeout time.Duration) (*pikpakgo.Task, error) {
	if od.client == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}

	log.Printf("⏳ 等待任务完成: %s (超时: %v)", taskId, timeout)

	task, err := od.client.WaitForOfflineDownloadComplete(taskId, timeout, func(task *pikpakgo.Task) {
		log.Printf("📊 下载进度: %s - %s (%.1f%%)", task.Name, task.Phase, task.Progress*100)
	})

	if err != nil {
		return nil, fmt.Errorf("等待任务完成失败: %v", err)
	}

	log.Printf("✅ 任务完成: %s", task.Name)
	return task, nil
}

// parseJSONFile 解析JSON配置文件
func parseJSONFile(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	return &config, nil
}

// login 登录函数（保持向后兼容）
func login(user, passwd string) (string, string) {
	client, err := pikpakgo.NewPikPakClient(user, passwd)
	if err != nil {
		log.Printf("❌ 创建客户端失败: %v", err)
		return user, passwd
	}

	err = client.Login()
	if err != nil {
		log.Printf("❌ 登录失败: %v", err)
		return user, passwd
	}

	log.Printf("✅ 登录成功: %s", user)
	return user, passwd
}

// OfflineDownload 使用新的下载器类的示例函数
func OfflineDownload(fileName, magnetLink string) {
	downloader, err := NewOfflineDownloader("config.json")
	if err != nil {
		log.Printf("❌ 创建下载器失败: %v", err)
		return
	}

	err = downloader.AddMagnetTask(fileName, magnetLink)
	if err != nil {
		log.Printf("❌ 添加下载任务失败: %v", err)
		return
	}

	log.Printf("✅ 下载任务添加成功")
}

// initializeTargetFolder 初始化目标文件夹
func (od *OfflineDownloader) initializeTargetFolder() error {
	// 如果已经指定了文件夹ID，直接使用
	if od.config.Pikpak.FolderID != "" {
		log.Printf("📁 使用指定的文件夹ID: %s", od.config.Pikpak.FolderID)
		return nil
	}

	// 如果指定了文件夹路径，尝试获取或创建
	if od.config.Pikpak.FolderPath != "" {
		log.Printf("📁 获取文件夹路径: %s", od.config.Pikpak.FolderPath)

		folderID, err := od.client.FolderPathToID(od.config.Pikpak.FolderPath, true)
		if err != nil {
			return fmt.Errorf("获取文件夹ID失败: %v", err)
		}

		// 更新配置中的文件夹ID
		od.config.Pikpak.FolderID = folderID
		log.Printf("✅ 文件夹ID获取成功: %s", folderID)
		return nil
	}

	// 如果都没有指定，使用根目录
	log.Printf("📁 使用根目录作为下载目标")
	od.config.Pikpak.FolderID = ""
	return nil
}

// getTargetFolderID 获取目标文件夹ID
func (od *OfflineDownloader) getTargetFolderID() string {
	return od.config.Pikpak.FolderID
}

// CreateDownloadFolder 创建下载文件夹
func (od *OfflineDownloader) CreateDownloadFolder(folderName string) (*pikpakgo.File, error) {
	if od.client == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}

	log.Printf("📁 创建文件夹: %s", folderName)

	parentID := od.getTargetFolderID()
	folder, err := od.client.CreateFolder(folderName, parentID)
	if err != nil {
		return nil, fmt.Errorf("创建文件夹失败: %v", err)
	}

	log.Printf("✅ 文件夹创建成功: %s (ID: %s)", folderName, folder.ID)
	return folder, nil
}

// ListFolderContents 列出文件夹内容
func (od *OfflineDownloader) ListFolderContents() ([]*pikpakgo.File, error) {
	if od.client == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}

	targetFolderID := od.getTargetFolderID()
	log.Printf("📁 获取文件夹内容: %s", targetFolderID)

	files, err := od.client.FileListAll(targetFolderID)
	if err != nil {
		return nil, fmt.Errorf("获取文件夹内容失败: %v", err)
	}

	log.Printf("✅ 文件夹内容获取完成，共 %d 个文件", len(files))
	for _, file := range files {
		log.Printf("   📄 %s (%s)", file.Name, file.Kind)
	}

	return files, nil
}
