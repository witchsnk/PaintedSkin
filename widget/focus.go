package widget

// FocusManager 负责 Tab 遍历与焦点归属（docs/06）。
//
// - 树序收集所有 Focusable && 有效可见 && 有效可用 的控件，成环；
// - Dialog 打开时切换为焦点陷阱（只在其子树内循环），关闭后由 Root 恢复；
// - 点击聚焦由 Root.Dispatch 在 Press 时执行；
// - 聚焦变化触发旧焦点 OnBlur、新焦点 OnFocus（各自打脏重绘）。
type FocusManager struct {
	root    Widget
	focused Widget
	trap    Widget
}

// NewFocusManager 创建焦点管理器。
func NewFocusManager() *FocusManager { return &FocusManager{} }

// SetRoot 设置遍历的根（通常为 Root 容器自身）。
func (m *FocusManager) SetRoot(r Widget) { m.root = r }

// Focused 返回当前焦点（可能为 nil）。
func (m *FocusManager) Focused() Widget { return m.focused }

// scope 返回当前遍历域：陷阱优先，否则整树。
func (m *FocusManager) scope() Widget {
	if m.trap != nil {
		return m.trap
	}
	return m.root
}

// Focusables 按树序返回当前域内可聚焦控件。
func (m *FocusManager) Focusables() []Widget {
	var out []Widget
	collectFocusables(m.scope(), &out)
	return out
}

func collectFocusables(w Widget, out *[]Widget) {
	if w == nil {
		return
	}
	if f, ok := w.(interface{ IsFocusable() bool }); ok && f.IsFocusable() {
		*out = append(*out, w)
	}
	for _, c := range w.Children() {
		collectFocusables(c, out)
	}
}

// Focus 把焦点给 w（nil 清空）。不可聚焦/陷阱外目标被忽略。
func (m *FocusManager) Focus(w Widget) {
	if w != nil {
		f, ok := w.(interface{ IsFocusable() bool })
		if !ok || !f.IsFocusable() {
			return
		}
		if m.trap != nil && !within(w, m.trap) {
			return
		}
	}
	if m.focused == w {
		return
	}
	if m.focused != nil {
		notifyBlur(m.focused)
	}
	m.focused = w
	if w != nil {
		notifyFocus(w)
	}
}

// FocusNext 向前移动焦点（环形）。
func (m *FocusManager) FocusNext() {
	m.step(1)
}

// FocusPrev 向后移动焦点（环形）。
func (m *FocusManager) FocusPrev() {
	m.step(-1)
}

func (m *FocusManager) step(d int) {
	list := m.Focusables()
	if len(list) == 0 {
		return
	}
	cur := -1
	for i, w := range list {
		if w == m.focused {
			cur = i
			break
		}
	}
	next := list[(cur+d+len(list))%len(list)]
	m.Focus(next)
}

// SetTrap 进入焦点陷阱（Dialog 入栈时由 Root 调用）。
func (m *FocusManager) SetTrap(w Widget) { m.trap = w }

// ClearTrap 离开焦点陷阱。
func (m *FocusManager) ClearTrap() { m.trap = nil }

// Trap 返回当前陷阱（测试用）。
func (m *FocusManager) Trap() Widget { return m.trap }

// within 报告 w 是否在 root 子树内（含 root 自身）。
func within(w, root Widget) bool {
	if w == nil || root == nil {
		return false
	}
	if w == root {
		return true
	}
	for _, c := range root.Children() {
		if within(w, c) {
			return true
		}
	}
	return false
}
