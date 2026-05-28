package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ResultQueue 本地结果队列
// API 不可用时将任务结果持久化到本地文件，API 恢复后自动重放
type ResultQueue struct {
	mu       sync.Mutex
	dir      string         // 队列文件目录
	maxSize  int            // 最大文件数量
	replayFn func(ctx context.Context, req *TaskResultReq) error // 重放函数
	stopChan chan struct{}
	stopOnce sync.Once
	logger   func(level, format string, args ...interface{})
}

// queuedResult 队列中的结果条目
type queuedResult struct {
	EnqueueTime time.Time      `json:"enqueueTime"`
	Request     *TaskResultReq `json:"request"`
}

// NewResultQueue 创建结果队列
func NewResultQueue(dir string, maxSize int, replayFn func(ctx context.Context, req *TaskResultReq) error) *ResultQueue {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &ResultQueue{
		dir:      dir,
		maxSize:  maxSize,
		replayFn: replayFn,
		stopChan: make(chan struct{}),
	}
}

// SetLogger 设置日志回调
func (q *ResultQueue) SetLogger(logger func(level, format string, args ...interface{})) {
	q.logger = logger
}

func (q *ResultQueue) log(level, format string, args ...interface{}) {
	if q.logger != nil {
		q.logger(level, format, args...)
	}
}

// Start 启动队列，创建目录并启动重放协程
func (q *ResultQueue) Start(ctx context.Context) error {
	if err := os.MkdirAll(q.dir, 0755); err != nil {
		return fmt.Errorf("create result queue dir: %w", err)
	}
	go q.replayLoop(ctx)
	return nil
}

// Stop 停止队列
func (q *ResultQueue) Stop() {
	q.stopOnce.Do(func() {
		close(q.stopChan)
	})
}

// Enqueue 将失败的结果入队到本地文件
func (q *ResultQueue) Enqueue(req *TaskResultReq) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 检查队列大小，超出时丢弃最旧的
	files := q.listFilesLocked()
	if len(files) >= q.maxSize {
		if len(files) > 0 {
			os.Remove(filepath.Join(q.dir, files[0]))
			q.log("WARN", "Result queue full, dropped oldest entry: %s", files[0])
		}
	}

	entry := queuedResult{
		EnqueueTime: time.Now(),
		Request:     req,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal queued result: %w", err)
	}

	suffix := req.MainTaskId
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	filename := fmt.Sprintf("%d_%s.json", time.Now().UnixMilli(), suffix)
	path := filepath.Join(q.dir, filename)

	// 原子写入：先写临时文件再重命名
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write queued result: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename queued result: %w", err)
	}

	q.log("INFO", "Queued result for task %s (%d assets) to %s", req.MainTaskId, len(req.Assets), filename)
	return nil
}

// replayLoop 定期检查并重放队列中的结果
func (q *ResultQueue) replayLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-q.stopChan:
			return
		case <-ticker.C:
			q.replayAll(ctx)
		}
	}
}

// replayAll 重放队列中的所有结果
func (q *ResultQueue) replayAll(ctx context.Context) {
	q.mu.Lock()
	files := q.listFilesLocked()
	q.mu.Unlock()

	if len(files) == 0 {
		return
	}

	q.log("INFO", "Replaying %d queued results...", len(files))

	for _, filename := range files {
		select {
		case <-ctx.Done():
			return
		case <-q.stopChan:
			return
		default:
		}

		path := filepath.Join(q.dir, filename)
		if err := q.replayOne(ctx, path); err != nil {
			q.log("WARN", "Failed to replay %s: %v, will retry later", filename, err)
			return // 保留剩余文件，下次重试
		}
		// 成功后删除文件
		os.Remove(path)
		q.log("INFO", "Replayed and removed %s", filename)
	}
}

// replayOne 重放单个结果文件
func (q *ResultQueue) replayOne(ctx context.Context, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var entry queuedResult
	if err := json.Unmarshal(data, &entry); err != nil {
		// 无法解析的文件直接删除
		q.log("WARN", "Corrupted queue file %s, removing", filepath.Base(path))
		return nil
	}

	replayCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	return q.replayFn(replayCtx, entry.Request)
}

// listFilesLocked 列出队列目录中的文件（需要持有锁）
func (q *ResultQueue) listFilesLocked() []string {
	entries, err := os.ReadDir(q.dir)
	if err != nil {
		return nil
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files
}

// Size 获取队列中的文件数量
func (q *ResultQueue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.listFilesLocked())
}
