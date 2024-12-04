package player

import (
	"FMgo/internal/logger"
	"fmt"
	"sync/atomic"
)

// Player 是播放器的主控制器，负责管理音频流的播放
type Player struct {
	streamPlayer *StreamPlayer
	isPlaying    atomic.Bool
	currentURL   string
}

// NewPlayer 创建一个新的播放器实例
func NewPlayer() (*Player, error) {
	logger.Info("初始化播放器")
	streamPlayer, err := NewStreamPlayer()
	if err != nil {
		logger.Error("创建流播放器失败: %v", err)
		return nil, fmt.Errorf("创建流播放器失败: %v", err)
	}

	return &Player{
		streamPlayer: streamPlayer,
		isPlaying:    atomic.Bool{},
	}, nil
}

// Play 开始播放指定的URL
func (p *Player) Play(url string) error {
	logger.Info("开始播放: %s", url)

	// 如果正在播放同一个URL，不做任何操作
	if p.isPlaying.Load() && p.currentURL == url {
		logger.Info("已经在播放该URL")
		return nil
	}

	p.Stop()
	// 开始新的播放
	if err := p.streamPlayer.PlayStream(url); err != nil {
		logger.Error("开始播放失败: %v", err)
		return fmt.Errorf("开始播放失败: %v", err)
	}

	p.isPlaying.Store(true)
	p.currentURL = url
	return nil
}

// Stop 停止当前播放
func (p *Player) Stop() {
	if p.isPlaying.Load() {
		p.streamPlayer.Stop()
		p.isPlaying.Store(false)
		p.currentURL = ""
	}
}

// IsPlaying 返回当前是否正在播放
func (p *Player) IsPlaying() bool {
	return p.isPlaying.Load()
}

// CurrentURL 返回当前正在播放的URL
func (p *Player) CurrentURL() string {
	return p.currentURL
}

// Cleanup 清理播放器资源
func (p *Player) Cleanup() {
	p.Stop()
	if p.streamPlayer != nil {
		p.streamPlayer.Cleanup()
	}
}
