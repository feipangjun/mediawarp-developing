package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/AkimioJR/MediaWarp/constants"
	"github.com/AkimioJR/MediaWarp/internal/config"
	"github.com/AkimioJR/MediaWarp/internal/handler"
	"github.com/AkimioJR/MediaWarp/internal/logging"
	"github.com/AkimioJR/MediaWarp/internal/router"
	"github.com/AkimioJR/MediaWarp/internal/service"
	"github.com/AkimioJR/MediaWarp/utils"

	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var (
	isDebug     bool   // 开启调试模式
	showVersion bool   // 显示版本信息
	configPath  string // 配置文件路径
)

func init() {
	gin.SetMode(gin.ReleaseMode)

	flag.BoolVar(&showVersion, "version", false, "显示版本信息")
	flag.BoolVar(&isDebug, "debug", false, "是否启用调试模式")
	flag.StringVar(&configPath, "config", "config/config.yaml", "指定配置文件路径")
	flag.Parse()

	fmt.Print(constants.LOGO)
	fmt.Println(utils.Center(fmt.Sprintf(" MediaWarp %s ", config.Version().AppVersion), 71, "="))
}

// 检测 FontInAss 服务状态
func checkFontInAssStatus() {
	if !config.Subtitle.FontInAss.Enable {
		logging.Info("FontInAss 服务: 未启用")
		return
	}

	if config.Subtitle.FontInAss.Addr == "" {
		logging.Warning("FontInAss 服务: 已启用但地址未配置")
		return
	}

	client := &http.Client{
		Timeout: 5 * time.Second, // 使用较短的超时时间进行检测
	}

	healthURL := config.Subtitle.FontInAss.Addr + "/api/mediawarp/subset"

	// 发送 GET 请求检测服务状态（FontInAss 不支持 HEAD 方法）
	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		logging.Warning("FontInAss 服务: 创建请求失败 - " + err.Error())
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		logging.Warning("FontInAss 服务: 连接失败 - " + err.Error())
		return
	}
	defer resp.Body.Close()

	// 检查服务状态：200-299 表示正常，405 表示方法不允许但服务可用
	if resp.StatusCode >= 200 && resp.StatusCode < 300 || resp.StatusCode == http.StatusMethodNotAllowed {
		logging.Info("FontInAss 服务: 正常运行 (地址: " + config.Subtitle.FontInAss.Addr + ")")
	} else {
		logging.Warning("FontInAss 服务: 响应异常 (状态码: " + strconv.Itoa(resp.StatusCode) + ")")
	}
}

func main() {
	if showVersion {
		versionInfo, _ := json.MarshalIndent(config.Version(), "", "  ")
		fmt.Println(string(versionInfo))
		return
	}

	if isDebug {
		logging.SetLevel(logrus.DebugLevel)
		logging.Info("已启用调试模式")
	}

	signChan := make(chan os.Signal, 1)
	errChan := make(chan error, 1)
	signal.Notify(signChan, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		logging.Info("MediaWarp 已退出")
	}()

	logging.Info("正在加载配置文件: ", configPath)

	if err := config.Init(configPath); err != nil { // 初始化配置
		panic("配置初始化失败: " + err.Error())
	}

	logging.Init()                                                                           // 初始化日志
	logging.Infof("上游媒体服务器类型：%s，服务器地址：%s", config.MediaServer.Type, config.MediaServer.ADDR) // 日志打印
	service.InitAlistClient()                                                                // 初始化Alist服务器
	if err := handler.Init(); err != nil {                                                   // 初始化媒体服务器处理器
		panic("媒体服务器处理器初始化失败: " + err.Error())
	}

	// 检测 FontInAss 服务状态
	checkFontInAssStatus()

	logging.Info("MediaWarp 监听端口：", config.Port)
	ginR := router.InitRouter() // 路由初始化
	logging.Info("MediaWarp 启动成功")
	go func() {
		if err := ginR.Run(config.ListenAddr()); err != nil {
			errChan <- err
		}
	}()

	select {
	case sig := <-signChan:
		logging.Info("MediaWarp 正在退出，信号：", sig)
	case err := <-errChan:
		logging.Error("MediaWarp 运行出错：", err)
	}
}
