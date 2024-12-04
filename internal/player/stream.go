package player

import (
	"FMgo/internal/config"
	"FMgo/internal/logger"
	"bufio"
	"container/ring"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type StreamPlayer struct {
	bufferFile string
	currentCmd *exec.Cmd
	stopChan   chan struct{}
	urlCache   *ring.Ring
	urlSet     map[string]bool
}

func NewStreamPlayer() (*StreamPlayer, error) {
	return &StreamPlayer{
		stopChan: make(chan struct{}),
		urlCache: ring.New(10),
		urlSet:   make(map[string]bool),
	}, nil
}

func (s *StreamPlayer) parseM3U8(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取 M3U8 失败: %v", err)
	}
	defer resp.Body.Close()

	var aacURLs []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, ".aac") {
			aacURLs = append(aacURLs, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("扫描 M3U8 失败: %v", err)
	}

	return aacURLs, nil
}

func (s *StreamPlayer) addToURLCache(url string) bool {
	if s.urlSet[url] {
		return false
	}

	if s.urlCache.Value != nil {
		delete(s.urlSet, s.urlCache.Value.(string))
	}

	s.urlCache.Value = url
	s.urlSet[url] = true
	s.urlCache = s.urlCache.Next()

	return true
}

func (s *StreamPlayer) downloadAndAppendAAC(url string, bufferFile string) error {
	if !s.addToURLCache(url) {
		return nil
	}

	logger.Debug("开始下载新的 URL: %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("下载 AAC 失败: %v", err)
	}
	defer resp.Body.Close()

	file, err := os.OpenFile(bufferFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开缓冲文件失败: %v", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("写入缓冲文件失败: %v", err)
	}

	return nil
}

func (s *StreamPlayer) PlayStream(url string) error {
	s.Stop()

	// 创建固定的缓冲文件
	bufferFile := filepath.Join(config.TempDir, "stream-buffer.aac")
	if err := os.WriteFile(bufferFile, nil, 0644); err != nil {
		return fmt.Errorf("创建缓冲文件失败: %v", err)
	}
	s.bufferFile = bufferFile
	s.stopChan = make(chan struct{})

	// 启动下载协程
	go func() {
		for {
			select {
			case <-s.stopChan:
				return
			default:
				aacURLs, err := s.parseM3U8(url)
				if err != nil {
					logger.Error("解析 M3U8 失败: %v", err)
					time.Sleep(time.Second * 5)
					continue
				}

				for _, aacURL := range aacURLs {
					select {
					case <-s.stopChan:
						return
					default:
						if err := s.downloadAndAppendAAC(aacURL, bufferFile); err != nil {
							logger.Error("下载和追加 AAC 失败: %v", err)
							continue
						}
					}
				}

				select {
				case <-s.stopChan:
					return
				case <-time.After(time.Second * 5):
				}
			}
		}
	}()

	// 启动播放协程
	go func() {
		// 等待文件可用，最多重试5次
		for i := 0; i < 5; i++ {
			info, err := os.Stat(bufferFile)
			if err == nil && info.Size() > 0 {
				cmd := exec.Command("afplay", bufferFile)
				s.currentCmd = cmd
				if err := cmd.Run(); err != nil && !strings.Contains(err.Error(), "signal: killed") {
					logger.Error("播放失败: %v", err)
				}
				return
			}
			logger.Info("等待文件可用，重试次数: %d", i+1)
			time.Sleep(time.Second)
		}
		logger.Error("等待文件可用超时，重试次数已用完")
	}()

	return nil
}

func (s *StreamPlayer) Stop() {
	if s.stopChan != nil {
		close(s.stopChan)
		s.stopChan = make(chan struct{})
	}

	if s.currentCmd != nil && s.currentCmd.Process != nil {
		s.currentCmd.Process.Kill()
		s.currentCmd = nil
	}

	s.urlCache = ring.New(10)
	s.urlSet = make(map[string]bool)

	// 清空缓冲文件
	if s.bufferFile != "" {
		os.Truncate(s.bufferFile, 0)
	}
}

func (s *StreamPlayer) Cleanup() {
	s.Stop()
	if s.bufferFile != "" {
		os.Remove(s.bufferFile)
		s.bufferFile = ""
	}
}
