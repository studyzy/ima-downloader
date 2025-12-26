# IMA.QQ.COM 下载器

一个命令行工具，用于批量下载 ima.qq.com 知识库中的文件。

## 功能特性

- ✅ 解析抓包的 HTTP 请求头
- ✅ **分页下载**：自动处理多页数据，每页20个文件
- ✅ **智能 Cursor 生成**：自动生成 Base64 编码的分页游标
- ✅ 批量下载文件到本地
- ✅ **日志记录**：详细记录下载进度、页数和失败信息到 `download.log`
- ✅ 自动创建下载目录
- ✅ 自动处理文件名中的非法字符

## 使用方法

### 1. 编译程序

```bash
go build -o ima-downloader
```

### 2. 运行程序

```bash
./ima-downloader
```

### 3. 输入抓包内容

程序启动后，会提示你输入抓包到的 HTTP 请求。你需要：

1. 打开浏览器开发者工具（F12）
2. 访问 ima.qq.com 知识库
3. 在 Network 标签中找到 `get_knowledge_list` 请求
4. 复制完整的 HTTP 请求内容（包括 Headers 和 Body）
5. 一次性粘贴到程序中
6. **粘贴完成后，按两次回车（输入两个空行）表示输入结束**

### 输入示例

```
POST /cgi-bin/knowledge_tab_reader/get_knowledge_list HTTP/1.1
Host: ima.qq.com
Connection: keep-alive
Content-Length: 161
sec-ch-ua-platform: "macOS"
from_browser_ima: 1
sec-ch-ua: "Not)A;Brand";v="8", "Chromium";v="138"
x-ima-bkn: 72264047
sec-ch-ua-mobile: ?0
traceparent: 00-2c48015142770d58e59e1b48c8beddac-7f1bf1619974bb6a-01
x-ima-cookie: PLATFORM=H5; CLIENT-TYPE=256020; WEB-VERSION=4.8.7; IMA-GUID=7298634324395740; ...
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 ...
accept: application/json
extension_version: 4.8.7
content-type: application/json
Origin: chrome-extension://nkohmbngmopdajidckglcoehlaeepeoi
Sec-Fetch-Site: none
Sec-Fetch-Mode: cors
Sec-Fetch-Dest: empty
Accept-Encoding: gzip, deflate, br, zstd
Accept-Language: en-US,en;q=0.9

{"sort_type":0,"need_default_cover":true,"knowledge_base_id":"7299678081146934","folder_id":"7299678081146934","cursor":"","limit":50,"version":"","ext_info":{}}

```

**注意**：Headers 和 Body 之间有一个空行，粘贴完成后再输入一个空行（共两个空行）表示结束。

### 4. 等待下载完成

程序会自动：
1. 解析你输入的 HTTP 请求
2. **分页获取**知识库文件列表（每页20个文件）
3. 逐个获取文件下载链接
4. 下载文件到 `downloads` 目录
5. 记录详细日志到 `download.log` 文件
6. 自动处理所有分页，直到下载完所有文件

## 输出示例

```
=== IMA.QQ.COM 下载器 ===

请粘贴抓包到的HTTP请求（包含Headers和Body），粘贴完成后输入两个空行结束：

POST /cgi-bin/knowledge_tab_reader/get_knowledge_list HTTP/1.1
Host: ima.qq.com
Connection: keep-alive
...（所有Headers）

{"sort_type":0,"need_default_cover":true,...}

（再输入一个空行结束）

✓ 已解析 18 个Header字段
✓ URL: https://ima.qq.com/cgi-bin/knowledge_tab_reader/get_knowledge_list
✓ Body: {"sort_type":0,"need_default_cover":true,...}

正在获取第 1 页（从索引 0 开始）...
成功获取 20 个文件（总共 150 个文件）

[1] 正在处理: 文件1.pdf
  ✓ 下载成功: downloads/文件1.pdf
[2] 正在处理: 文件2.pdf
  ✓ 下载成功: downloads/文件2.pdf
...

正在获取第 2 页（从索引 20 开始）...
成功获取 20 个文件（总共 150 个文件）

[21] 正在处理: 文件21.pdf
  ✓ 下载成功: downloads/文件21.pdf
...

已到达最后一页

========== 下载完成 ==========
总页数: 8
成功下载: 148 个文件
下载失败: 2 个文件
详细日志请查看: download.log
```

## 注意事项

1. 确保你的 HTTP 请求包含有效的认证信息（Cookie、Token等）
2. 下载的文件会保存在 `downloads` 目录中
3. 如果文件名包含非法字符，会自动替换为下划线
4. 程序会跳过下载失败的文件，继续处理下一个
5. **分页下载**：程序会自动处理所有分页，每页20个文件
6. **日志文件**：所有下载记录（包括成功和失败）都会保存在 `download.log` 中
7. 程序会在每页之间自动延迟500ms，避免请求过快

## 依赖

- Go 1.16 或更高版本
- 仅使用 Go 标准库，无需额外依赖

## 许可证

MIT License
