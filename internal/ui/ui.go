package ui

import (
	"FMgo/internal/logger"
	"fmt"
	"strings"
	"sync"

	"FMgo/internal/db"
	"FMgo/internal/model"
	"FMgo/internal/player"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

const (
	colorTitle       = ui.ColorGreen
	colorText        = ui.ColorWhite
	colorHighlight   = ui.ColorYellow
	colorBorder      = ui.ColorBlue
	colorSelected    = ui.ColorCyan
	colorCategory    = ui.ColorYellow
	colorStatusOK    = ui.ColorGreen
	colorStatusError = ui.ColorRed

	defaultStatus = "按 '/' 搜索 | 'h' 历史 | 'f' 收藏 | 'a' 收藏/取消 | 'q' 退出 | 's' 停止 | '?' 帮助 | ↑↓ 选择 | Enter 播放"
)

type UI struct {
	categories    []model.Category
	player        *player.Player
	db            *db.Database
	grid          *ui.Grid
	radioList     *widgets.List
	statusBar     *widgets.Paragraph
	searchInput   *widgets.Paragraph
	isSearching   bool
	searchText    string
	currentView   string // "main", "history", "favorites"
	mu            sync.RWMutex
	collapsedCats map[string]bool
}

func New(categories []model.Category, player *player.Player, db *db.Database) (*UI, error) {
	if err := ui.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize termui: %v", err)
	}

	u := &UI{
		categories:    categories,
		player:        player,
		db:            db,
		collapsedCats: make(map[string]bool),
		currentView:   "main",
	}

	// 初始化所有分类为折叠状态
	for _, cat := range categories {
		u.collapsedCats[cat.Name] = true
	}

	u.setupWidgets()
	u.updateRadioList(true)
	return u, nil
}

func (u *UI) setupWidgets() {
	u.radioList = widgets.NewList()
	u.radioList.Title = "电台列表"
	u.radioList.TextStyle = ui.NewStyle(colorText)
	u.radioList.SelectedRowStyle = ui.NewStyle(colorSelected, ui.ColorBlack, ui.ModifierBold)
	u.radioList.BorderStyle = ui.NewStyle(colorBorder)
	u.radioList.TitleStyle = ui.NewStyle(colorTitle, ui.ColorClear, ui.ModifierBold)
	u.radioList.WrapText = false
	u.radioList.PaddingLeft = 2
	u.radioList.PaddingRight = 2

	u.searchInput = widgets.NewParagraph()
	u.searchInput.Title = "搜索"
	u.searchInput.Text = ""
	u.searchInput.BorderStyle = ui.NewStyle(colorBorder)
	u.searchInput.TitleStyle = ui.NewStyle(colorTitle, ui.ColorClear, ui.ModifierBold)
	u.searchInput.PaddingLeft = 2
	u.searchInput.PaddingRight = 2

	u.statusBar = widgets.NewParagraph()
	u.statusBar.Title = "状态"
	u.statusBar.Text = defaultStatus
	u.statusBar.BorderStyle = ui.NewStyle(colorBorder)
	u.statusBar.TitleStyle = ui.NewStyle(colorTitle, ui.ColorClear, ui.ModifierBold)
	u.statusBar.TextStyle = ui.NewStyle(colorText)
	u.statusBar.PaddingLeft = 2
	u.statusBar.PaddingRight = 2

	u.grid = ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	u.grid.SetRect(0, 0, termWidth, termHeight)

	u.grid.Set(
		ui.NewRow(0.2, u.searchInput),
		ui.NewRow(0.6, u.radioList),
		ui.NewRow(0.2, u.statusBar),
	)
}

func (u *UI) updateRadioList(isFlushRow bool) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if u.currentView == "history" {
		u.showHistory()
		return
	}

	if u.currentView == "favorites" {
		u.showFavorites()
		return
	}

	var items []string
	for _, cat := range u.categories {
		collapsed := u.collapsedCats[cat.Name]
		indicator := "▶"
		if !collapsed {
			indicator = "▼"
		}
		catName := fmt.Sprintf("[%s %s](fg:cyan)", indicator, cat.Name)
		items = append(items, catName)

		if !collapsed {
			for _, radio := range cat.RadioList {
				items = append(items, fmt.Sprintf(" •%s", radio.Name))
			}
		}
	}

	u.radioList.Title = "电台列表"
	u.radioList.Rows = items
	if isFlushRow {
		u.radioList.SelectedRow = 0
	}
	ui.Render(u.grid)
}

func (u *UI) showHistory() {
	history, err := u.db.GetHistory(20)
	if err != nil {
		u.setStatus(fmt.Sprintf("加载历史记录失败: %v", err), colorStatusError)
		return
	}

	var historyItems []string
	historyItems = append(historyItems, "[播放历史](fg:yellow)")

	for _, h := range history {
		historyItems = append(historyItems, fmt.Sprintf(" •%s", h.RadioName))
	}

	if len(history) == 0 {
		historyItems = append(historyItems, "  暂无播放记录")
	}

	u.radioList.Title = "播放历史"
	u.radioList.Rows = historyItems
	u.radioList.SelectedRow = 1 // 从第一个历史记录开始
	ui.Render(u.grid)
}

func (u *UI) showFavorites() {
	favorites, err := u.db.GetFavorites()
	if err != nil {
		u.setStatus(fmt.Sprintf("加载收藏失败: %v", err), colorStatusError)
		return
	}

	var items []string
	items = append(items, "[收藏列表](fg:yellow)")

	for _, radio := range favorites {
		items = append(items, fmt.Sprintf(" •%s", radio.Name))
	}

	if len(items) == 1 {
		items = append(items, "  暂无收藏")
	}

	u.radioList.Title = "收藏列表"
	u.radioList.Rows = items
	u.radioList.SelectedRow = 1 // 从第一个收藏开始
	ui.Render(u.grid)
}

func (u *UI) toggleCategory(name string) {
	// 去除可能的前缀和后缀
	name = strings.TrimSpace(name)
	name = strings.TrimRight(name, "](fg:cyan)")
	name = strings.TrimLeft(name, "[▶")
	name = strings.TrimLeft(name, "[▼")
	name = strings.TrimSpace(name)
	if _, exists := u.collapsedCats[name]; exists {
		u.collapsedCats[name] = !u.collapsedCats[name]
		u.updateRadioList(false)
	}
}

func (u *UI) toggleFavorite(name string) {
	name = strings.TrimPrefix(name, " •")

	// 查找电台信息
	var radio model.Radio
	found := false
	for _, cat := range u.categories {
		for _, r := range cat.RadioList {
			if r.Name == name {
				radio = r
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		u.setStatus(fmt.Sprintf("未找到电台: %s", name), colorStatusError)
		return
	}

	// 检查是否已收藏
	isFav, err := u.db.IsFavorite(name)
	if err != nil {
		u.setStatus(fmt.Sprintf("检查收藏状态失败: %v", err), colorStatusError)
		return
	}

	if isFav {
		// 取消收藏
		if err := u.db.RemoveFavorite(name); err != nil {
			u.setStatus(fmt.Sprintf("取消收藏失败: %v", err), colorStatusError)
			return
		}
		u.setStatus(fmt.Sprintf("已取消收藏: %s", name), colorStatusOK)
	} else {
		// 添加收藏
		if err := u.db.AddFavorite(radio); err != nil {
			u.setStatus(fmt.Sprintf("添加收藏失败: %v", err), colorStatusError)
			return
		}
		u.setStatus(fmt.Sprintf("已添加收藏: %s", name), colorStatusOK)
	}

	// 如果当前在收藏列表视图，刷新显示
	if u.currentView == "favorites" {
		u.showFavorites()
	}
}

func (u *UI) setStatus(status string, color ui.Color) {
	u.statusBar.TextStyle = ui.NewStyle(color)
	u.statusBar.Text = status
	ui.Render(u.grid)
}

func (u *UI) findAndPlayRadio(name string) bool {
	name = strings.TrimPrefix(name, " •")
	logger.Info("查找电台: %s", name)

	for _, cat := range u.categories {
		for _, radio := range cat.RadioList {
			if radio.Name == name {
				logger.Info("播放电台: %s, URL: %s", radio.Name, radio.PlayURL)
				if err := u.player.Play(radio.PlayURL); err != nil {
					u.setStatus(fmt.Sprintf("播放错误: %v", err), colorStatusError)
					return false
				}
				if err := u.db.AddHistory(radio); err != nil {
					logger.Error("记录历史失败: %v", err)
				}
				u.setStatus(fmt.Sprintf("正在播放: %s", radio.Name), colorStatusOK)
				return true
			}
		}
	}
	u.setStatus(fmt.Sprintf("未找到电台: %s", name), colorStatusError)
	return false
}

func (u *UI) enterSearchMode() {
	u.isSearching = true
	u.searchText = ""
	u.searchInput.Text = "搜索: "
	u.updateSearchResults()
}

func (u *UI) exitSearchMode() {
	u.isSearching = false
	u.searchText = ""
	u.searchInput.Text = ""
	u.updateRadioList(true)
}

func (u *UI) handleSearchMode(e ui.Event) {
	switch e.ID {
	case "<Escape>":
		u.exitSearchMode()
	case "<Backspace>":
		if len(u.searchText) > 0 {
			// 处理中文字符的退格
			r := []rune(u.searchText)
			u.searchText = string(r[:len(r)-1])
			u.searchInput.Text = fmt.Sprintf("搜索: %s", u.searchText)
			u.updateSearchResults()
		}
	case "<Space>":
		u.searchText += " "
		u.searchInput.Text = fmt.Sprintf("搜索: %s", u.searchText)
		u.updateSearchResults()
	case "j", "<Down>":
		if len(u.radioList.Rows) > 1 { // 考虑标题行
			if u.radioList.SelectedRow < len(u.radioList.Rows)-1 {
				u.radioList.ScrollDown()
			}
		}
	case "k", "<Up>":
		if len(u.radioList.Rows) > 1 { // 考虑标题行
			if u.radioList.SelectedRow > 1 { // 不要选到标题行
				u.radioList.ScrollUp()
			}
		}
	default:
		if len(e.ID) == 1 || len([]rune(e.ID)) == 1 { // 支持中文输入
			u.searchText += e.ID
			u.searchInput.Text = fmt.Sprintf("搜索: %s", u.searchText)
			u.updateSearchResults()
		}
	}
	ui.Render(u.grid)
}

func (u *UI) updateSearchResults() {
	var items []string
	searchText := strings.ToLower(u.searchText)

	if searchText == "" {
		u.updateRadioList(true)
		return
	}

	items = append(items, "[搜索结果](fg:yellow)")
	for _, cat := range u.categories {
		for _, radio := range cat.RadioList {
			if strings.Contains(strings.ToLower(radio.Name), searchText) {
				items = append(items, fmt.Sprintf(" •%s", radio.Name))
			}
		}
	}

	if len(items) == 1 {
		items = append(items, "  未找到匹配结果")
	}

	u.radioList.Title = "搜索结果"
	u.radioList.Rows = items
	u.radioList.SelectedRow = 1 // 从第一个搜索结果开始
	ui.Render(u.grid)
}

func (u *UI) Run() {
	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		case "?":
			if !u.isSearching {
				u.setStatus(defaultStatus, colorText)
			}
		case "<Enter>":
			if u.isSearching {
				if len(u.radioList.Rows) == 0 {
					continue
				}
				selected := u.radioList.Rows[u.radioList.SelectedRow]

				if strings.HasPrefix(selected, " •") {
					// 暂时注释掉播放逻辑
					u.findAndPlayRadio(selected)
					logger.Info("搜索结果中选中电台: %s", selected)
					u.exitSearchMode()
				}
				continue
			}

			if u.currentView == "history" {
				// 暂时注释掉播放逻辑
				u.findAndPlayRadio(u.radioList.Rows[u.radioList.SelectedRow])
				logger.Info("选中历史记录: %s", u.radioList.Rows[u.radioList.SelectedRow])
				u.currentView = "main"
				u.updateRadioList(true)
				continue
			}

			if u.currentView == "favorites" {
				// 暂时注释掉播放逻辑
				u.findAndPlayRadio(u.radioList.Rows[u.radioList.SelectedRow])
				logger.Info("选中收藏电台: %s", u.radioList.Rows[u.radioList.SelectedRow])
				u.currentView = "main"
				u.updateRadioList(true)
				continue
			}

			if strings.HasPrefix(u.radioList.Rows[u.radioList.SelectedRow], " •") {
				// 暂时注释掉播放逻辑
				u.findAndPlayRadio(u.radioList.Rows[u.radioList.SelectedRow])
				logger.Info("选中电台: %s", u.radioList.Rows[u.radioList.SelectedRow])
			} else {
				// 折叠/展开分类
				u.toggleCategory(u.radioList.Rows[u.radioList.SelectedRow])
			}
		case "h":
			if !u.isSearching {
				u.currentView = "history"
				u.showHistory()
			}
		case "f":
			if !u.isSearching {
				u.currentView = "favorites"
				u.showFavorites()
			}
		case "/":
			u.enterSearchMode()
		case "s":
			if u.player.IsPlaying() {
				u.player.Stop()
				u.setStatus("播放已停止", colorText)
			}
		case "a":
			if !u.isSearching && len(u.radioList.Rows) > 0 {
				selected := u.radioList.Rows[u.radioList.SelectedRow]
				if strings.HasPrefix(selected, " •") {
					u.toggleFavorite(selected)
				}
			}
		case "<Resize>":
			termWidth, termHeight := ui.TerminalDimensions()
			u.grid.SetRect(0, 0, termWidth, termHeight)
			ui.Render(u.grid)
		default:
			if u.isSearching {
				u.handleSearchMode(e)
				continue
			}
			switch e.ID {
			case "j", "<Down>":
				if len(u.radioList.Rows) > 0 {
					if u.radioList.SelectedRow < len(u.radioList.Rows)-1 {
						u.radioList.ScrollDown()
					}
				}
			case "k", "<Up>":
				if len(u.radioList.Rows) > 0 {
					if u.radioList.SelectedRow > 0 {
						u.radioList.ScrollUp()
					}
				}
			case "<Tab>":
				if !u.isSearching {
					switch u.currentView {
					case "main":
						u.currentView = "history"
						u.showHistory()
					case "history":
						u.currentView = "favorites"
						u.showFavorites()
					case "favorites":
						u.currentView = "main"
						u.updateRadioList(true)
					}
				}
			}
		}
		ui.Render(u.grid)
	}
}

func (u *UI) Close() {
	if u.player != nil {
		u.player.Cleanup()
	}
	ui.Close()
}
