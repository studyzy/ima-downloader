package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HTTPHeaderInfo 存储解析后的HTTP请求信息
type HTTPHeaderInfo struct {
	URL     string
	Headers map[string]string
	Body    string
}

// KnowledgeListResponse 知识库列表响应
type KnowledgeListResponse struct {
	Code          int    `json:"code"`
	Msg           string `json:"msg"`
	KnowledgeList []struct {
		Title          string `json:"title"`
		ParentFolderID string `json:"parent_folder_id"`
		MediaID        string `json:"media_id"`
	} `json:"knowledge_list"`
	NextCursor string `json:"next_cursor"`
	IsEnd      bool   `json:"is_end"`
	TotalSize  string `json:"total_size"`
}

// Logger 日志记录器
var logger *log.Logger

// MediaResponse 媒体文件响应
type MediaResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	JumpURL string `json:"jump_url"`
	Title   string `json:"title"`
}

func main() {
	fmt.Println("=== IMA.QQ.COM 下载器 ===")
	fmt.Println()

	// 初始化日志
	if err := initLogger(); err != nil {
		fmt.Printf("错误：初始化日志失败 - %v\n", err)
		return
	}
	logger.Println("========== 开始新的下载任务 ==========")

	// 读取用户输入的HTTP请求
	headerInfo, err := readHTTPRequest()
	if err != nil {
		fmt.Printf("错误：解析HTTP请求失败 - %v\n", err)
		logger.Printf("错误：解析HTTP请求失败 - %v", err)
		return
	}

	// 创建下载目录
	downloadDir := "downloads"
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		fmt.Printf("错误：创建下载目录失败 - %v\n", err)
		logger.Printf("错误：创建下载目录失败 - %v", err)
		return
	}

	// 解析初始Body以获取知识库ID和limit
	var initialBody map[string]interface{}
	if err := json.Unmarshal([]byte(headerInfo.Body), &initialBody); err != nil {
		fmt.Printf("错误：解析Body失败 - %v\n", err)
		logger.Printf("错误：解析Body失败 - %v", err)
		return
	}

	// 开始分页下载
	pageSize := 20 // 每页20个文件
	currentIndex := 0
	pageNum := 1
	totalDownloaded := 0
	totalFailed := 0

	for {
		// 更新Body中的cursor和limit
		initialBody["cursor"] = generateCursor(currentIndex)
		initialBody["limit"] = pageSize

		bodyBytes, _ := json.Marshal(initialBody)
		headerInfo.Body = string(bodyBytes)

		fmt.Printf("\n正在获取第 %d 页（从索引 %d 开始）...\n", pageNum, currentIndex)
		logger.Printf("正在获取第 %d 页（从索引 %d 开始）", pageNum, currentIndex)

		knowledgeList, err := getKnowledgeList(headerInfo)
		if err != nil {
			fmt.Printf("错误：获取知识库列表失败 - %v\n", err)
			logger.Printf("错误：获取第 %d 页失败 - %v", pageNum, err)
			break
		}

		fmt.Printf("成功获取 %d 个文件（总共 %s 个文件）\n\n", len(knowledgeList.KnowledgeList), knowledgeList.TotalSize)
		logger.Printf("第 %d 页：获取到 %d 个文件", pageNum, len(knowledgeList.KnowledgeList))

		if len(knowledgeList.KnowledgeList) == 0 {
			fmt.Println("本页没有文件，下载完成")
			logger.Println("本页没有文件，下载完成")
			break
		}

		// 遍历每个文件并下载
		for i, item := range knowledgeList.KnowledgeList {
			globalIndex := totalDownloaded + i + 1
			fmt.Printf("[%d] 正在处理: %s\n", globalIndex, item.Title)

			// 获取下载链接
			downloadURL, err := getMediaDownloadURL(headerInfo, item.ParentFolderID, item.MediaID)
			if err != nil {
				fmt.Printf("  ✗ 获取下载链接失败: %v\n", err)
				logger.Printf("[失败] 第%d页 文件%d: %s - 获取下载链接失败: %v", pageNum, i+1, item.Title, err)
				totalFailed++
				continue
			}

			// 下载文件
			filePath := filepath.Join(downloadDir, sanitizeFilename(item.Title))
			if err := downloadFile(downloadURL, filePath); err != nil {
				fmt.Printf("  ✗ 下载失败: %v\n", err)
				logger.Printf("[失败] 第%d页 文件%d: %s - 下载失败: %v", pageNum, i+1, item.Title, err)
				totalFailed++
				continue
			}

			fmt.Printf("  ✓ 下载成功: %s\n", filePath)
			logger.Printf("[成功] 第%d页 文件%d: %s", pageNum, i+1, item.Title)
		}

		totalDownloaded += len(knowledgeList.KnowledgeList)

		// 检查是否还有下一页
		if knowledgeList.IsEnd {
			fmt.Println("\n已到达最后一页")
			logger.Println("已到达最后一页")
			break
		}

		// 更新索引和页码
		currentIndex += len(knowledgeList.KnowledgeList)
		pageNum++

		// 短暂延迟，避免请求过快
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("\n========== 下载完成 ==========\n")
	fmt.Printf("总页数: %d\n", pageNum)
	fmt.Printf("成功下载: %d 个文件\n", totalDownloaded-totalFailed)
	fmt.Printf("下载失败: %d 个文件\n", totalFailed)
	fmt.Printf("详细日志请查看: download.log\n")

	logger.Printf("========== 下载任务完成 ==========")
	logger.Printf("总页数: %d, 成功: %d, 失败: %d", pageNum, totalDownloaded-totalFailed, totalFailed)
}

// initLogger 初始化日志记录器
func initLogger() error {
	logFile, err := os.OpenFile("download.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	logger = log.New(logFile, "", log.LstdFlags)
	return nil
}

// generateCursor 生成Base64编码的cursor
// 格式：0x08 + Index值（变长编码）
func generateCursor(index int) string {
	if index == 0 {
		return ""
	}
	
	// 使用变长编码（Varint）
	// 对于小于128的数字，直接使用一个字节
	// 对于大于等于128的数字，使用多字节编码
	var buf []byte
	buf = append(buf, 0x08) // 固定前缀
	
	// Varint编码
	for index >= 0x80 {
		buf = append(buf, byte(index)|0x80)
		index >>= 7
	}
	buf = append(buf, byte(index))
	
	return base64.StdEncoding.EncodeToString(buf)
}

// readHTTPRequest 读取并解析用户输入的HTTP请求
func readHTTPRequest() (*HTTPHeaderInfo, error) {
	fmt.Println("请粘贴抓包到的HTTP请求（包含Headers和Body），粘贴完成后输入两个空行结束：")
	fmt.Println()
	
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string
	emptyLineCount := 0
	
	// 读取所有行，直到连续遇到两个空行
	for scanner.Scan() {
		line := scanner.Text()
		
		if line == "" {
			emptyLineCount++
			// 连续两个空行表示输入结束
			if emptyLineCount >= 2 {
				break
			}
			lines = append(lines, line)
		} else {
			emptyLineCount = 0
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(lines) == 0 {
		return nil, fmt.Errorf("未输入任何内容")
	}

	info := &HTTPHeaderInfo{
		Headers: make(map[string]string),
	}

	// 解析第一行获取URL
	firstLine := lines[0]
	parts := strings.Fields(firstLine)
	if len(parts) >= 2 {
		path := parts[1]
		info.URL = "https://ima.qq.com" + path
	}

	// 查找空行位置（Headers和Body的分隔符）
	emptyLineIndex := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "" {
			emptyLineIndex = i
			break
		}
	}

	// 解析Headers（从第二行到空行之前）
	headerEndIndex := len(lines)
	if emptyLineIndex > 0 {
		headerEndIndex = emptyLineIndex
	}

	for i := 1; i < headerEndIndex; i++ {
		line := lines[i]
		colonIndex := strings.Index(line, ":")
		if colonIndex > 0 {
			key := strings.TrimSpace(line[:colonIndex])
			value := strings.TrimSpace(line[colonIndex+1:])
			info.Headers[key] = value
		}
	}

	// 解析Body（空行之后的内容）
	if emptyLineIndex > 0 && emptyLineIndex+1 < len(lines) {
		// 将空行后的所有内容合并为Body
		bodyLines := lines[emptyLineIndex+1:]
		info.Body = strings.TrimSpace(strings.Join(bodyLines, ""))
	}

	// 验证Body是否是有效的JSON
	if info.Body != "" {
		var jsonTest interface{}
		if err := json.Unmarshal([]byte(info.Body), &jsonTest); err != nil {
			return nil, fmt.Errorf("Body不是有效的JSON格式: %v", err)
		}
	}

	fmt.Printf("✓ 已解析 %d 个Header字段\n", len(info.Headers))
	fmt.Printf("✓ URL: %s\n", info.URL)
	fmt.Printf("✓ Body: %s\n\n", info.Body)

	return info, nil
}

// getKnowledgeList 获取知识库列表
func getKnowledgeList(headerInfo *HTTPHeaderInfo) (*KnowledgeListResponse, error) {
	req, err := http.NewRequest("POST", headerInfo.URL, bytes.NewBufferString(headerInfo.Body))
	if err != nil {
		return nil, err
	}

	// 设置Headers
	for key, value := range headerInfo.Headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result KnowledgeListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API返回错误: code=%d, msg=%s", result.Code, result.Msg)
	}

	return &result, nil
}

// getMediaDownloadURL 获取媒体文件的下载链接
func getMediaDownloadURL(headerInfo *HTTPHeaderInfo, knowledgeBaseID, mediaID string) (string, error) {
	url := "https://ima.qq.com/cgi-bin/file_manager/get_media"

	// 构造请求Body
	requestBody := map[string]interface{}{
		"knowledge_base_id": knowledgeBaseID,
		"media_id":          mediaID,
		"scene":             1,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	// 设置Headers
	for key, value := range headerInfo.Headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result MediaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("API返回错误: code=%d, msg=%s", result.Code, result.Msg)
	}

	return result.JumpURL, nil
}

// downloadFile 下载文件到本地
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// sanitizeFilename 清理文件名，移除不合法字符
func sanitizeFilename(filename string) string {
	// 替换不合法的文件名字符
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(filename)
}
