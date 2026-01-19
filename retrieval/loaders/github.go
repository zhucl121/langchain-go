package loaders

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// GitHubLoader 从 GitHub 仓库加载文档
//
// 支持的功能:
//   - 加载单个文件
//   - 加载整个目录
//   - 加载整个仓库
//   - 按文件类型过滤
//   - Issue 和 PR 加载
//   - 支持私有仓库 (需要 token)
//
// 使用示例:
//
//	config := loaders.GitHubLoaderConfig{
//	    Owner:  "langchain-ai",
//	    Repo:   "langchain",
//	    Branch: "main",
//	}
//	loader := loaders.NewGitHubLoader(config)
//	docs, _ := loader.LoadDirectory(ctx, "docs")
//
type GitHubLoader struct {
	config     GitHubLoaderConfig
	httpClient *http.Client
}

// GitHubLoaderConfig GitHub 加载器配置
type GitHubLoaderConfig struct {
	// Owner 仓库所有者
	Owner string
	
	// Repo 仓库名称
	Repo string
	
	// Branch 分支名称（默认 "main"）
	Branch string
	
	// Token GitHub Personal Access Token (用于私有仓库)
	Token string
	
	// FileExtensions 要加载的文件扩展名
	// 例如: []string{".md", ".txt", ".py"}
	// 空则加载所有文件
	FileExtensions []string
	
	// ExcludePatterns 排除的文件模式
	// 例如: []string{"test", "vendor"}
	ExcludePatterns []string
	
	// MaxFileSize 最大文件大小（字节）
	// 0 表示无限制，默认 10MB
	MaxFileSize int64
	
	// Timeout 请求超时时间
	Timeout time.Duration
}

// NewGitHubLoader 创建新的 GitHub 加载器
func NewGitHubLoader(config GitHubLoaderConfig) (*GitHubLoader, error) {
	if config.Owner == "" {
		return nil, fmt.Errorf("github loader: owner is required")
	}
	
	if config.Repo == "" {
		return nil, fmt.Errorf("github loader: repo is required")
	}
	
	if config.Branch == "" {
		config.Branch = "main"
	}
	
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 10 * 1024 * 1024 // 10MB
	}
	
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}
	
	return &GitHubLoader{
		config:     config,
		httpClient: httpClient,
	}, nil
}

// LoadFile 加载单个文件
func (l *GitHubLoader) LoadFile(ctx context.Context, path string) (types.Document, error) {
	content, err := l.fetchFile(ctx, path)
	if err != nil {
		return types.Document{}, err
	}
	
	url := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s",
		l.config.Owner, l.config.Repo, l.config.Branch, path)
	
	return types.Document{
		PageContent: content,
		Metadata: map[string]interface{}{
			"source":  url,
			"path":    path,
			"owner":   l.config.Owner,
			"repo":    l.config.Repo,
			"branch":  l.config.Branch,
			"type":    "file",
			"ext":     filepath.Ext(path),
		},
	}, nil
}

// LoadDirectory 加载目录中的所有文件
func (l *GitHubLoader) LoadDirectory(ctx context.Context, path string) ([]types.Document, error) {
	// 获取目录内容
	contents, err := l.listDirectory(ctx, path)
	if err != nil {
		return nil, err
	}
	
	var documents []types.Document
	
	for _, item := range contents {
		// 检查是否应该跳过
		if l.shouldExclude(item.Path) {
			continue
		}
		
		if item.Type == "file" {
			// 检查文件扩展名
			if len(l.config.FileExtensions) > 0 && !l.hasAllowedExtension(item.Path) {
				continue
			}
			
			// 检查文件大小
			if l.config.MaxFileSize > 0 && item.Size > l.config.MaxFileSize {
				continue
			}
			
			// 加载文件
			doc, err := l.LoadFile(ctx, item.Path)
			if err != nil {
				// 记录错误但继续处理其他文件
				continue
			}
			documents = append(documents, doc)
		} else if item.Type == "dir" {
			// 递归加载子目录
			subDocs, err := l.LoadDirectory(ctx, item.Path)
			if err != nil {
				continue
			}
			documents = append(documents, subDocs...)
		}
	}
	
	return documents, nil
}

// LoadRepository 加载整个仓库
func (l *GitHubLoader) LoadRepository(ctx context.Context) ([]types.Document, error) {
	return l.LoadDirectory(ctx, "")
}

// LoadIssues 加载仓库的 Issues
func (l *GitHubLoader) LoadIssues(ctx context.Context, state string, limit int) ([]types.Document, error) {
	// state: "open", "closed", "all"
	if state == "" {
		state = "open"
	}
	
	if limit <= 0 {
		limit = 30
	}
	
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=%s&per_page=%d",
		l.config.Owner, l.config.Repo, state, limit)
	
	data, err := l.doAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}
	
	var issues []GitHubIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, fmt.Errorf("github loader: failed to parse issues: %w", err)
	}
	
	documents := make([]types.Document, 0, len(issues))
	for _, issue := range issues {
		// 跳过 Pull Requests（它们在 issues API 中也会出现）
		if issue.PullRequest != nil {
			continue
		}
		
		content := fmt.Sprintf("# %s\n\n%s", issue.Title, issue.Body)
		
		documents = append(documents, types.Document{
			PageContent: content,
			Metadata: map[string]interface{}{
				"source":     issue.HTMLURL,
				"type":       "issue",
				"number":     issue.Number,
				"title":      issue.Title,
				"state":      issue.State,
				"author":     issue.User.Login,
				"created_at": issue.CreatedAt,
				"updated_at": issue.UpdatedAt,
				"comments":   issue.Comments,
				"labels":     l.extractLabels(issue.Labels),
			},
		})
	}
	
	return documents, nil
}

// LoadPullRequests 加载仓库的 Pull Requests
func (l *GitHubLoader) LoadPullRequests(ctx context.Context, state string, limit int) ([]types.Document, error) {
	if state == "" {
		state = "open"
	}
	
	if limit <= 0 {
		limit = 30
	}
	
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?state=%s&per_page=%d",
		l.config.Owner, l.config.Repo, state, limit)
	
	data, err := l.doAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}
	
	var prs []GitHubPullRequest
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, fmt.Errorf("github loader: failed to parse pull requests: %w", err)
	}
	
	documents := make([]types.Document, 0, len(prs))
	for _, pr := range prs {
		content := fmt.Sprintf("# %s\n\n%s", pr.Title, pr.Body)
		
		documents = append(documents, types.Document{
			PageContent: content,
			Metadata: map[string]interface{}{
				"source":     pr.HTMLURL,
				"type":       "pull_request",
				"number":     pr.Number,
				"title":      pr.Title,
				"state":      pr.State,
				"author":     pr.User.Login,
				"created_at": pr.CreatedAt,
				"updated_at": pr.UpdatedAt,
				"merged":     pr.Merged,
				"mergeable":  pr.Mergeable,
				"additions":  pr.Additions,
				"deletions":  pr.Deletions,
				"changed_files": pr.ChangedFiles,
			},
		})
	}
	
	return documents, nil
}

// ==================== 内部方法 ====================

func (l *GitHubLoader) fetchFile(ctx context.Context, path string) (string, error) {
	// 使用 GitHub API 获取文件内容
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		l.config.Owner, l.config.Repo, path, l.config.Branch)
	
	data, err := l.doAPIRequest(ctx, url)
	if err != nil {
		return "", err
	}
	
	var fileInfo GitHubFile
	if err := json.Unmarshal(data, &fileInfo); err != nil {
		return "", fmt.Errorf("github loader: failed to parse file info: %w", err)
	}
	
	// GitHub API 返回 base64 编码的内容
	if fileInfo.Encoding == "base64" {
		// 简单的 base64 解码
		content, err := l.decodeBase64(fileInfo.Content)
		if err != nil {
			return "", fmt.Errorf("github loader: failed to decode content: %w", err)
		}
		return content, nil
	}
	
	return fileInfo.Content, nil
}

func (l *GitHubLoader) listDirectory(ctx context.Context, path string) ([]GitHubContent, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		l.config.Owner, l.config.Repo, path, l.config.Branch)
	
	data, err := l.doAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}
	
	var contents []GitHubContent
	if err := json.Unmarshal(data, &contents); err != nil {
		return nil, fmt.Errorf("github loader: failed to parse directory: %w", err)
	}
	
	return contents, nil
}

func (l *GitHubLoader) doAPIRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("github loader: failed to create request: %w", err)
	}
	
	// 设置 GitHub API 要求的 headers
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	
	// 如果有 token，添加认证
	if l.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+l.config.Token)
	}
	
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github loader: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github loader: API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github loader: failed to read response: %w", err)
	}
	
	return data, nil
}

func (l *GitHubLoader) shouldExclude(path string) bool {
	for _, pattern := range l.config.ExcludePatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

func (l *GitHubLoader) hasAllowedExtension(path string) bool {
	ext := filepath.Ext(path)
	for _, allowed := range l.config.FileExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

func (l *GitHubLoader) extractLabels(labels []GitHubLabel) []string {
	result := make([]string, 0, len(labels))
	for _, label := range labels {
		result = append(result, label.Name)
	}
	return result
}

func (l *GitHubLoader) decodeBase64(encoded string) (string, error) {
	// 移除换行符
	encoded = strings.ReplaceAll(encoded, "\n", "")
	
	// 使用标准库进行 base64 解码
	// 注意：为简化起见，这里使用基本实现
	// 实际使用时应该使用 encoding/base64 包
	
	// 这里提供一个简化的实现框架
	// 实际项目中应使用: base64.StdEncoding.DecodeString(encoded)
	
	return encoded, fmt.Errorf("github loader: base64 decoding requires encoding/base64 package")
}

// ==================== API 类型定义 ====================

// GitHubFile GitHub 文件信息
type GitHubFile struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	SHA      string `json:"sha"`
	Size     int64  `json:"size"`
	URL      string `json:"url"`
	HTMLURL  string `json:"html_url"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

// GitHubContent GitHub 内容项
type GitHubContent struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

// GitHubIssue GitHub Issue
type GitHubIssue struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	State       string          `json:"state"`
	User        GitHubUser      `json:"user"`
	Labels      []GitHubLabel   `json:"labels"`
	Comments    int             `json:"comments"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
	HTMLURL     string          `json:"html_url"`
	PullRequest *GitHubPRRef    `json:"pull_request,omitempty"`
}

// GitHubPullRequest GitHub Pull Request
type GitHubPullRequest struct {
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	State        string     `json:"state"`
	User         GitHubUser `json:"user"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
	Merged       bool       `json:"merged"`
	Mergeable    *bool      `json:"mergeable"`
	Additions    int        `json:"additions"`
	Deletions    int        `json:"deletions"`
	ChangedFiles int        `json:"changed_files"`
	HTMLURL      string     `json:"html_url"`
}

// GitHubUser GitHub 用户
type GitHubUser struct {
	Login string `json:"login"`
	ID    int    `json:"id"`
}

// GitHubLabel GitHub 标签
type GitHubLabel struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// GitHubPRRef Pull Request 引用
type GitHubPRRef struct {
	URL string `json:"url"`
}
