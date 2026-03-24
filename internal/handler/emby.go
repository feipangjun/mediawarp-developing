package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AkimioJR/MediaWarp/constants"
	"github.com/AkimioJR/MediaWarp/internal/config"
	"github.com/AkimioJR/MediaWarp/internal/logging"
	"github.com/AkimioJR/MediaWarp/internal/service/emby"
	"github.com/AkimioJR/MediaWarp/utils"

	"github.com/gin-gonic/gin"
)

// // 带引用计数的互斥锁
// type mutexWithRefCount struct {
// 	mu       sync.Mutex
// 	refCount int32 // 使用 atomic 操作
// }

// Emby服务器处理器
type EmbyHandler struct {
	client          *emby.Client           // Emby客户端
	routerRules     []RegexpRouteRule      // 正则路由规则
	proxy           *httputil.ReverseProxy // 反向代理
	httpStrmHandler StrmHandlerFunc
	// playbackInfoMutex sync.Map // 视频流处理并发控制，确保同一个 item ID 的重定向请求串行化，避免重复获取缓存
}

// 初始化
func NewEmbyServerHandler(addr string, apiKey string) (*EmbyHandler, error) {
	var handler = EmbyHandler{}
	handler.client = emby.New(addr, apiKey)
	target, err := url.Parse(handler.client.GetEndpoint())
	if err != nil {
		return nil, err
	}
	handler.proxy = httputil.NewSingleHostReverseProxy(target)

	{ // 初始化路由规则
		handler.routerRules = []RegexpRouteRule{
			{
				Regexp:  constants.EmbyRegexp.Router.VideosHandler,
				Handler: handler.VideosHandler,
			},
			{
				Regexp: constants.EmbyRegexp.Router.ModifyPlaybackInfo,
				Handler: responseModifyCreater(
					&httputil.ReverseProxy{Director: handler.proxy.Director},
					handler.ModifyPlaybackInfo,
				),
			},
			{
				Regexp: constants.EmbyRegexp.Router.ModifyBaseHtmlPlayer,
				Handler: responseModifyCreater(
					&httputil.ReverseProxy{Director: handler.proxy.Director},
					handler.ModifyBaseHtmlPlayer,
				),
			},
		}

		if config.Web.Enable {
			if config.Web.Index || config.Web.Head != "" || config.Web.ExternalPlayerUrl || config.Web.VideoTogether {
				handler.routerRules = append(handler.routerRules,
					RegexpRouteRule{
						Regexp: constants.EmbyRegexp.Router.ModifyIndex,
						Handler: responseModifyCreater(
							&httputil.ReverseProxy{Director: handler.proxy.Director},
							handler.ModifyIndex,
						),
					},
				)
			}
		}
		if config.Subtitle.Enable {
			// 添加字幕处理路由规则
			handler.routerRules = append(handler.routerRules,
				RegexpRouteRule{
					Regexp: constants.EmbyRegexp.Router.ModifySubtitles,
					Handler: responseModifyCreater(
						&httputil.ReverseProxy{Director: handler.proxy.Director},
						handler.ModifySubtitles,
					),
				},
			)

			// 添加内封字幕流处理路由规则（完全匹配 FontInAss 的 nginx 配置）
			if config.Subtitle.SRT2ASS || config.Subtitle.SubSet {
				handler.routerRules = append(handler.routerRules,
					RegexpRouteRule{
						// 匹配 FontInAss nginx 配置中的路径模式：/videos/(.*)/Subtitles/(.*)/(Stream.ass|Stream.ssa|Stream.srt|Stream.)
						Regexp: regexp.MustCompile(`(?i)^(/emby)?/videos/(.*)/Subtitles/(.*)/(Stream\.(ass|ssa|srt)|Stream\.)$`),
						Handler: responseModifyCreater(
							&httputil.ReverseProxy{Director: handler.proxy.Director},
							handler.ModifySubtitleStream,
						),
					},
				)
			}
		}
	}
	handler.httpStrmHandler, err = getHTTPStrmHandler()
	if err != nil {
		return nil, fmt.Errorf("创建 HTTPStrm 处理器失败: %w", err)
	}
	return &handler, nil
}

// 转发请求至上游服务器
func (handler *EmbyHandler) ReverseProxy(rw http.ResponseWriter, req *http.Request) {
	handler.proxy.ServeHTTP(rw, req)
}

// 正则路由表
func (handler *EmbyHandler) GetRegexpRouteRules() []RegexpRouteRule {
	return handler.routerRules
}

func (handler *EmbyHandler) GetImageCacheRegexp() *regexp.Regexp {
	return constants.EmbyRegexp.Cache.Image
}

func (handler *EmbyHandler) GetSubtitleCacheRegexp() *regexp.Regexp {
	return constants.EmbyRegexp.Cache.Subtitle
}

// 修改播放信息请求
//
// /Items/:itemId/PlaybackInfo
// 强制将 HTTPStrm 设置为支持直链播放和转码、AlistStrm 设置为支持直链播放并且禁止转码
func (handler *EmbyHandler) ModifyPlaybackInfo(rw *http.Response) error {
	startTime := time.Now()
	defer func() {
		logging.Debugf("处理 ModifyPlaybackInfo 耗时：%s", time.Since(startTime))
	}()

	defer rw.Body.Close()
	body, err := io.ReadAll(rw.Body)
	if err != nil {
		logging.Warning("读取 Body 出错：", err)
		return err
	}

	jsonChain := utils.NewJsonChainFromBytesWithCopy(body, jsonChainOption)

	var playbackInfoResponse emby.PlaybackInfoResponse
	if err = json.Unmarshal(body, &playbackInfoResponse); err != nil {
		logging.Warning("解析 emby.PlaybackInfoResponse Json 错误：", err)
		return err
	}

	for index, mediasource := range playbackInfoResponse.MediaSources {
		startTime := time.Now()

		logging.Debug("请求 ItemsServiceQueryItem：" + *mediasource.ID)
		itemResponse, err := handler.client.ItemsServiceQueryItem(strings.Replace(*mediasource.ID, "mediasource_", "", 1), 1, "Path,MediaSources") // 查询 item 需要去除前缀仅保留数字部分
		if err != nil {
			logging.Warning("请求 ItemsServiceQueryItem 失败：", err)
			continue
		}

		bsePath := "MediaSources." + strconv.Itoa(index) + "."
		item := itemResponse.Items[0]
		strmFileType, opt := recgonizeStrmFileType(*item.Path)
		switch strmFileType {
		case constants.HTTPStrm: // HTTPStrm 设置支持直链播放并且禁止转码
			processHTTPStrmPlaybackInfo(
				jsonChain,
				bsePath,
				*mediasource.ItemID,
				*mediasource.ID,
				mediasource.DirectStreamURL,
			)

		case constants.AlistStrm: // AlistStm 设置支持直链播放并且禁止转码
			processAlistStrmPlaybackInfo(
				jsonChain,
				bsePath,
				*mediasource.ItemID,
				*mediasource.ID,
				opt.(string),
				mediasource.DirectStreamURL,
				*item.Path,
				mediasource.Size,
			)
		}

		logging.Debugf("处理 %s 的 MediaSource %s 耗时：%s", *item.Path, *mediasource.ID, time.Since(startTime))
	}

	body, err = jsonChain.Result()
	if err != nil {
		logging.Warning("操作 emby.PlaybackInfoResponse Json 错误：", err)
		return err
	}

	rw.Header.Set("Content-Type", "application/json")        // 更新 Content-Type 头
	rw.Header.Set("Content-Length", strconv.Itoa(len(body))) // 更新 Content-Length 头
	rw.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}

// 视频流处理器
//
// 支持播放本地视频、重定向 HttpStrm、AlistStrm
func (handler *EmbyHandler) VideosHandler(ctx *gin.Context) {
	if ctx.Request.Method == http.MethodHead { // 不额外处理 HEAD 请求
		handler.ReverseProxy(ctx.Writer, ctx.Request)
		logging.Debug("VideosHandler 不处理 HEAD 请求，转发至上游服务器")
		return
	}

	orginalPath := ctx.Request.URL.Path
	matches := constants.EmbyRegexp.Others.VideoRedirectReg.FindStringSubmatch(orginalPath)
	if len(matches) == 2 {
		redirectPath := fmt.Sprintf("/videos/%s/stream", matches[0])
		logging.Debugf("%s 重定向至：%s", orginalPath, redirectPath)
		ctx.Redirect(http.StatusFound, redirectPath)
		return
	}

	// EmbyServer <= 4.8 ====> mediaSourceID = 343121
	// EmbyServer >= 4.9 ====> mediaSourceID = mediasource_31
	mediaSourceID := ctx.Query("mediasourceid")

	logging.Debugf("请求 ItemsServiceQueryItem：%s", mediaSourceID)
	mediaSourceID_without_prefix := strings.Replace(mediaSourceID, "mediasource_", "", 1)
	itemResponse, err := handler.client.ItemsServiceQueryItem(mediaSourceID_without_prefix, 1, "Path,MediaSources") // 查询 item 需要去除前缀仅保留数字部分
	if err != nil {
		logging.Warning("请求 ItemsServiceQueryItem 失败：", err)
		handler.ReverseProxy(ctx.Writer, ctx.Request)
		return
	}

	item := itemResponse.Items[0]

	if !strings.HasSuffix(strings.ToLower(*item.Path), ".strm") { // 不是 Strm 文件
		logging.Debug("播放本地视频：" + *item.Path + "，不进行处理")
		handler.ReverseProxy(ctx.Writer, ctx.Request)
		return
	}

	strmFileType, opt := recgonizeStrmFileType(*item.Path)

	for _, mediasource := range item.MediaSources {
		logging.Debugf("mediasource.ID: %s ; mediaSourceID: %s ; mediaSourceID_without_prefix: %s", *mediasource.ID, mediaSourceID, mediaSourceID_without_prefix)
		// EmbyServer >= 4.9 返回的ID带有前缀mediasource_
		if strings.Replace(*mediasource.ID, "mediasource_", "", 1) == mediaSourceID_without_prefix {
			switch strmFileType {
			case constants.HTTPStrm:
				if *mediasource.Protocol == emby.HTTP {
					ctx.Redirect(http.StatusFound, handler.httpStrmHandler(*mediasource.Path, ctx.Request.UserAgent()))
					return
				}

			case constants.AlistStrm: // 无需判断 *mediasource.Container 是否以Strm结尾，当 AlistStrm 存储的位置有对应的文件时，*mediasource.Container 会被设置为文件后缀
				res, err := alistStrmHandler(*mediasource.Path, opt.(string), false)
				if err != nil {
					logging.Warningf("获取 AlistStrm 重定向 URL 失败: %#v", err)
					handler.ReverseProxy(ctx.Writer, ctx.Request)
					return
				}
				ctx.Redirect(http.StatusFound, res.url)
				return

			case constants.UnknownStrm:
				handler.ReverseProxy(ctx.Writer, ctx.Request)
				return
			}
		}
	}
}

// 判断是否为 ASS 字幕
func isAssSubtitle(content []byte) bool {
	contentStr := string(content)
	// ASS 字幕通常包含特定的头部信息
	return strings.Contains(contentStr, "[Script Info]") &&
		strings.Contains(contentStr, "[V4+ Styles]") &&
		strings.Contains(contentStr, "[Events]")
}

// 发送字幕到 FontInAss 进行处理
func (handler *EmbyHandler) sendToFontInAss(subtitle []byte) ([]byte, error) {
	if !config.Subtitle.FontInAss.Enable || config.Subtitle.FontInAss.Addr == "" {
		return nil, fmt.Errorf("FontInAss 未启用或地址未配置")
	}

	client := &http.Client{
		Timeout: config.Subtitle.FontInAss.Timeout,
	}

	// 构建 FontInAss API 地址
	fontInAssURL := config.Subtitle.FontInAss.Addr + "/api/mediawarp/subset"

	// 发送 POST 请求到 FontInAss
	resp, err := client.Post(fontInAssURL, "application/octet-stream", bytes.NewReader(subtitle))
	if err != nil {
		return nil, fmt.Errorf("连接 FontInAss 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FontInAss 处理失败: %s, 响应: %s", resp.Status, string(body))
	}

	// 读取处理后的字幕数据
	processedSubtitle, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 FontInAss 响应失败: %w", err)
	}

	return processedSubtitle, nil
}

// 处理字幕流请求（内封字幕）
//
// 处理 /videos/(.*)/Subtitles/(.*)/(Stream.ass|Stream.ssa|Stream.srt|Stream.) 等路径
func (handler *EmbyHandler) ModifySubtitleStream(rw *http.Response) error {
	return handler.processSubtitleContent(rw, "字幕流")
}

// 修改字幕
//
// 将 SRT 字幕转 ASS，并支持 FontInAss 字体子集化
func (handler *EmbyHandler) ModifySubtitles(rw *http.Response) error {
	return handler.processSubtitleContent(rw, "字幕")
}

// 通用的字幕处理逻辑
func (handler *EmbyHandler) processSubtitleContent(rw *http.Response, subtitleType string) error {
	// 检查是否为 304 Not Modified 响应
	if rw.StatusCode == http.StatusNotModified {
		logging.Debug("字幕", subtitleType, "请求返回 304 Not Modified，跳过处理")
		return nil // 直接返回，不修改响应
	}

	defer rw.Body.Close()
	subtitle, err := io.ReadAll(rw.Body) // 读取字幕文件
	if err != nil {
		logging.Warning("读取原始", subtitleType, " Body 出错：", err)
		return err
	}

	// 添加调试信息：输出字幕大小和内容预览
	logging.Info("处理", subtitleType, "请求，字幕大小：", len(subtitle), "字节")
	if len(subtitle) > 0 && len(subtitle) < 1000 {
		logging.Debug("字幕内容预览：", string(subtitle[:min(len(subtitle), 200)]))
	}

	// 检查原始字幕类型
	isSRT := utils.IsSRT(subtitle)
	isASS := isAssSubtitle(subtitle)

	logging.Debug("字幕格式检测 - SRT:", isSRT, " ASS:", isASS)

	// 如果是 ASS 字幕，并且启用了字体子集化，发送到 FontInAss
	if isASS && config.Subtitle.SubSet && config.Subtitle.FontInAss.Enable {
		logging.Info("检测到原始 ASS", subtitleType, "，开始字体子集化处理")
		processedSubtitle, err := handler.sendToFontInAss(subtitle)
		if err == nil {
			subtitle = processedSubtitle
			logging.Info(subtitleType, "字体子集化处理完成，处理后大小：", len(subtitle), "字节")
		} else {
			logging.Warning("FontInAss 处理失败，使用原始", subtitleType, ":", err)
		}
	} else if isSRT && config.Subtitle.SRT2ASS {
		// 如果是 SRT 字幕，并且在 MediaWarp 中转成 ASS
		logging.Info(subtitleType, "文件为 SRT 格式，转为 ASS 格式")
		subtitle = utils.SRT2ASS(subtitle, config.Subtitle.ASSStyle)
		logging.Info("已将 SRT", subtitleType, "转为 ASS 格式")
	} else {
		// 其他情况：既不是 ASS 也不是 SRT，或者未启用相关功能
		if isSRT {
			logging.Debug(subtitleType, "为 SRT 格式，但未启用 SRT2ASS，跳过转换")
		} else if isASS {
			logging.Debug(subtitleType, "为 ASS 格式，但未启用字体子集化，跳过处理")
		} else {
			logging.Debug(subtitleType, "为其他格式，跳过处理")
		}
	}

	rw.Header.Set("Content-Length", strconv.Itoa(len(subtitle)))
	rw.Body = io.NopCloser(bytes.NewReader(subtitle))
	return nil
}

// 辅助函数：返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 修改 basehtmlplayer.js
//
// 用于修改播放器 JS，实现跨域播放 Strm 文件（302 重定向）
func (handler *EmbyHandler) ModifyBaseHtmlPlayer(rw *http.Response) error {
	defer rw.Body.Close()
	body, err := io.ReadAll(rw.Body)
	if err != nil {
		return err
	}

	body = bytes.ReplaceAll(body, []byte(`mediaSource.IsRemote&&"DirectPlay"===playMethod?null:"anonymous"`), []byte("null")) // 修改响应体
	rw.Header.Set("Content-Length", strconv.Itoa(len(body)))
	rw.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}

// 修改首页函数
func (handler *EmbyHandler) ModifyIndex(rw *http.Response) error {
	var (
		htmlFilePath string = path.Join(config.CostomDir(), "index.html")
		htmlContent  []byte
		addHEAD      bytes.Buffer
		err          error
	)

	defer rw.Body.Close()  // 无论哪种情况，最终都要确保原 Body 被关闭，避免内存泄漏
	if !config.Web.Index { // 从上游获取响应体
		if htmlContent, err = io.ReadAll(rw.Body); err != nil {
			return err
		}
	} else { // 从本地文件读取index.html
		if htmlContent, err = os.ReadFile(htmlFilePath); err != nil {
			logging.Warning("读取文件内容出错，错误信息：", err)
			return err
		}
	}

	if config.Web.Head != "" { // 用户自定义HEAD
		addHEAD.WriteString(config.Web.Head + "\n")
	}
	if config.Web.ExternalPlayerUrl { // 外部播放器
		addHEAD.WriteString(`<script src="/MediaWarp/static/embyExternalUrl/embyWebAddExternalUrl/embyLaunchPotplayer.js"></script>` + "\n")
	}
	if config.Web.Crx { // crx 美化
		addHEAD.WriteString(`<link rel="stylesheet" id="theme-css" href="/MediaWarp/static/emby-crx/static/css/style.css" type="text/css" media="all" />
    <script src="/MediaWarp/static/emby-crx/static/js/common-utils.js"></script>
    <script src="/MediaWarp/static/emby-crx/static/js/jquery-3.6.0.min.js"></script>
    <script src="/MediaWarp/static/emby-crx/static/js/md5.min.js"></script>
    <script src="/MediaWarp/static/emby-crx/content/main.js"></script>` + "\n")
	}
	if config.Web.ActorPlus { // 过滤没有头像的演员和制作人员
		addHEAD.WriteString(`<script src="/MediaWarp/static/emby-web-mod/actorPlus/actorPlus.js"></script>` + "\n")
	}
	if config.Web.FanartShow { // 显示同人图（fanart图）
		addHEAD.WriteString(`<script src="/MediaWarp/static/emby-web-mod/fanart_show/fanart_show.js"></script>` + "\n")
	}
	if config.Web.Danmaku { // 弹幕
		addHEAD.WriteString(`<script src="/MediaWarp/static/dd-danmaku/ede.js" defer></script>` + "\n")
	}
	if config.Web.VideoTogether { // VideoTogether
		addHEAD.WriteString(`<script src="https://2gether.video/release/extension.website.user.js"></script>` + "\n")
	}
	addHEAD.WriteString(`<!-- MediaWarp Web 页面修改功能 -->` + "\n" + "</head>")
	htmlContent = bytes.Replace(htmlContent, []byte("</head>"), addHEAD.Bytes(), 1) // 将添加HEAD
	rw.Header.Set("Content-Length", strconv.Itoa(len(htmlContent)))
	rw.Body = io.NopCloser(bytes.NewReader(htmlContent))
	return nil
}

var _ MediaServerHandler = (*EmbyHandler)(nil) // 确保 EmbyHandler 实现 MediaServerHandler 接口
