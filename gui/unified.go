package gui

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/coyim/gotk3adapter/gtki"
	"github.com/coyim/gotk3adapter/pangoi"
)

const (
	ulIndexID          = 0
	ulIndexDisplayName = 1
	ulIndexJid         = 2
	ulIndexColor       = 3
	ulIndexBackground  = 4
	ulIndexWeight      = 5
	ulIndexTooltip     = 6
	ulIndexStatusIcon  = 7
	ulIndexUnderline   = 8
)

var ulAllIndexValues = []int{0, 1, 2, 3, 4, 5, 6, 7, 8}

type unifiedLayout struct {
	ui             *gtkUI
	cl             *conversationList
	headerBar      gtki.HeaderBar
	closeButton    gtki.Button
	minimizeButton gtki.Button
	maximizeButton gtki.Button
	leftPane       gtki.Box
	rightPane      gtki.Box
	notebook       gtki.Notebook
	header         gtki.Label
	headerBox      gtki.Box
	close          gtki.Button
	convsVisible   bool
	inPageSet      bool
	isFullscreen   bool
	itemMap        map[int]*conversationStackItem
}

type conversationList struct {
	layout *unifiedLayout
	view   gtki.TreeView
	model  gtki.ListStore
}

type conversationStackItem struct {
	*conversationPane
	pageIndex      int
	needsAttention bool
	iter           gtki.TreeIter
	layout         *unifiedLayout
}

func newUnifiedLayout(ui *gtkUI, left, parent gtki.Box) *unifiedLayout {
	ul := &unifiedLayout{
		ui:           ui,
		cl:           &conversationList{},
		leftPane:     left,
		itemMap:      make(map[int]*conversationStackItem),
		isFullscreen: false,
	}
	ul.cl.layout = ul

	builder := newBuilder("UnifiedLayout")
	builder.getItems(
		"treeview", &ul.cl.view,
		"liststore", &ul.cl.model,
		"headerbar", &ul.headerBar,
		"close-button", &ul.closeButton,
		"minimize-button", &ul.minimizeButton,
		"maximize-button", &ul.maximizeButton,
		"right", &ul.rightPane,
		"notebook", &ul.notebook,
		"header_label", &ul.header,
		"header_box", &ul.headerBox,
		"close_button", &ul.close,
	)

	//ul.cl.model needs to be kept beyond the lifespan of the builder.
	ul.cl.model.Ref()
	runtime.SetFinalizer(ul.cl, func(cl interface{}) {
		cl.(*conversationList).model.Unref()
		cl.(*conversationList).model = nil
	})

	builder.ConnectSignals(map[string]interface{}{
		"on_activate":        ul.cl.onActivate,
		"on_clicked":         ul.onCloseClicked,
		"on_close_button":    ui.quit,
		"on_minimize_button": ui.minimize,
		"on_maximize_button": ui.maximize,
		"on_switch_page":     ul.onSwitchPage,
	})

	connectShortcut("<Primary>Page_Down", ul.ui.window, ul.nextTab)
	connectShortcut("<Primary>Page_Up", ul.ui.window, ul.previousTab)
	connectShortcut("F11", ul.ui.window, ul.toggleFullscreen)

	parent.PackStart(ul.rightPane, false, true, 0)
	parent.SetChildPacking(ul.leftPane, false, true, 0, gtki.PACK_START)

	ul.rightPane.Hide()

	ul.headerBar.SetShowCloseButton(false)
	closeImg, _ := g.gtk.ImageNewFromIconName("window-close-symbolic", gtki.ICON_SIZE_MENU)
	ul.closeButton.SetImage(closeImg)
	minImg, _ := g.gtk.ImageNewFromIconName("window-minimize-symbolic", gtki.ICON_SIZE_MENU)
	ul.minimizeButton.SetImage(minImg)
	maxImg, _ := g.gtk.ImageNewFromIconName("window-maximize-symbolic", gtki.ICON_SIZE_MENU)
	ul.maximizeButton.SetImage(maxImg)

	ul.ui.window.SetTitlebar(ul.headerBar)

	left.SetHAlign(gtki.ALIGN_FILL)
	left.SetHExpand(true)
	return ul
}

func (ul *unifiedLayout) createConversation(account *account, uid, resource string, existing *conversationPane) conversationView {
	cp := createConversationPane(account, uid, ul.ui, ul.ui.window)
	if existing != nil {
		b, _ := existing.history.GetBuffer()
		cp.history.SetBuffer(b)
	}
	cp.connectEnterHandler(nil)
	idx := ul.notebook.AppendPage(cp.widget, nil)
	if idx < 0 {
		panic("Failed to append page to notebook")
	}

	csi := &conversationStackItem{
		conversationPane: cp,
		pageIndex:        idx,
		layout:           ul,
	}

	//	csi.entry.SetHasFrame(true)
	csi.entryScroll.SetMarginTop(5)
	csi.entryScroll.SetMarginBottom(5)

	tabLabel := csi.shortName()
	if resource != "" {
		tabLabel = tabLabel + " [at] " + resource
	}
	ul.notebook.SetTabLabelText(cp.widget, tabLabel)
	ul.itemMap[idx] = csi
	buffer, _ := csi.history.GetBuffer()
	buffer.Connect("changed", func() {
		ul.onConversationChanged(csi)
	})
	return csi
}

func (ul *unifiedLayout) onConversationChanged(csi *conversationStackItem) {
	if !csi.isCurrent() {
		csi.needsAttention = true
		csi.applyTextWeight()
	}
}

func (cl *conversationList) add(csi *conversationStackItem) {
	if csi.iter == nil {
		csi.iter = cl.model.Append()
		cl.updateItem(csi)
	}
}

func (cl *conversationList) remove(csi *conversationStackItem) {
	if csi.iter == nil {
		return
	}
	cl.model.Remove(csi.iter)
	csi.iter = nil
}

func (cl *conversationList) updateItem(csi *conversationStackItem) {
	cs := cl.layout.ui.currentColorSet()
	peer, ok := csi.currentPeer()
	if !ok {
		log.Printf("No peer found for %s", csi.to)
		return
	}
	cl.model.Set2(csi.iter, ulAllIndexValues, []interface{}{
		csi.pageIndex,
		csi.shortName(),
		peer.Jid,
		decideColorFor(cs, peer),
		cs.rosterPeerBackground,
		csi.getTextWeight(),
		createTooltipFor(peer),
		statusIcons[decideStatusFor(peer)].getPixbuf(),
		csi.getUnderline(),
	},
	)
}

func (ul *unifiedLayout) showConversations() {
	if ul.convsVisible {
		return
	}

	ul.leftPane.SetHExpand(false)
	ul.rightPane.SetHExpand(true)

	ul.ui.window.Resize(934, 600)

	ul.rightPane.Show()

	ul.convsVisible = true
	ul.update()
}

func (ul *unifiedLayout) hideConversations() {
	if !ul.convsVisible {
		return
	}

	width := ul.leftPane.GetAllocatedWidth()
	height := ul.ui.window.GetAllocatedHeight()
	ul.rightPane.SetHExpand(false)
	ul.rightPane.Hide()
	ul.leftPane.SetHExpand(true)
	ul.ui.window.Resize(width, height)
	ul.convsVisible = false
	ul.headerBar.SetSubtitle("")
}

func (csi *conversationStackItem) isVisible() bool {
	return csi.isCurrent() && csi.layout.ui.window.HasToplevelFocus()
}

func (csi *conversationStackItem) setEnabled(enabled bool) {
	log.Printf("csi.SetEnabled(%v)", enabled)
}

func (csi *conversationStackItem) shortName() string {
	ss := strings.Split(csi.to, "@")
	uiName := ss[0]

	peer, ok := csi.layout.ui.getPeer(csi.account, csi.to)
	if ok && peer.NameForPresentation() != peer.Jid {
		uiName = peer.NameForPresentation()
	}

	return uiName
}

func (csi *conversationStackItem) isCurrent() bool {
	if csi == nil {
		return false
	}
	return csi.layout.notebook.GetCurrentPage() == csi.pageIndex
}

func (csi *conversationStackItem) getUnderline() int {
	if csi.isCurrent() {
		return pangoi.UNDERLINE_SINGLE
	}
	return pangoi.UNDERLINE_NONE
}

func (csi *conversationStackItem) getTextWeight() int {
	if csi.needsAttention {
		return 700
	}
	return 500
}

func (csi *conversationStackItem) applyTextWeight() {
	if csi.iter == nil {
		return
	}
	weight := csi.getTextWeight()
	csi.layout.cl.model.SetValue(csi.iter, ulIndexWeight, weight)
}

func (csi *conversationStackItem) show(userInitiated bool) {
	csi.layout.showConversations()
	csi.updateSecurityWarning()
	csi.layout.cl.add(csi)
	csi.widget.Show()
	if userInitiated {
		csi.bringToFront()
		return
	}
	if !csi.isCurrent() {
		csi.needsAttention = true
		csi.applyTextWeight()
	}
}

func (csi *conversationStackItem) bringToFront() {
	peer, _ := csi.currentPeer()
	csi.layout.showConversations()
	csi.needsAttention = false
	csi.applyTextWeight()
	csi.layout.setCurrentPage(csi)
	title := fmt.Sprintf("%s <-> %s", csi.account.session.DisplayName(), peer.NameForPresentation())
	csi.layout.header.SetText(title)
	csi.layout.headerBar.SetSubtitle(title)
	csi.entry.GrabFocus()
	csi.layout.update()
}

func (csi *conversationStackItem) remove() {
	csi.layout.cl.remove(csi)
	csi.widget.Hide()
}

func (csi *conversationStackItem) destroy() {
	csi.layout.closeNthConversation(csi.pageIndex)
	csi.widget.Destroy()
}

func (cl *conversationList) getItemForIter(iter gtki.TreeIter) *conversationStackItem {
	val, err := cl.model.GetValue(iter, ulIndexID)
	if err != nil {
		log.Printf("Error getting ulIndexID value: %v", err)
		return nil
	}
	gv, err := val.GoValue()
	if err != nil {
		log.Printf("Error getting GoValue for ulIndexID: %v", err)
		return nil
	}
	return cl.layout.itemMap[gv.(int)]
}

func (cl *conversationList) onActivate(v gtki.TreeView, path gtki.TreePath) {
	iter, err := cl.model.GetIter(path)
	if err != nil {
		log.Printf("Error converting path to iter: %v", err)
		return
	}
	csi := cl.getItemForIter(iter)
	if csi != nil {
		csi.bringToFront()
		cl.removeSelection()
	}
}

func (cl *conversationList) removeSelection() {
	ts, _ := cl.view.GetSelection()
	if _, iter, ok := ts.GetSelected(); ok {
		path, _ := cl.model.GetPath(iter)
		ts.UnselectPath(path)
	}
}

func (ul *unifiedLayout) setCurrentPage(csi *conversationStackItem) {
	ul.inPageSet = true
	ul.notebook.SetCurrentPage(csi.pageIndex)
	ul.update()
	ul.inPageSet = false
}

func (ul *unifiedLayout) closeNthConversation(n int) {
	if n >= 0 {
		item := ul.itemMap[n]
		if item != nil {
			item.remove()
		}
	}

	if !ul.displayFirstConvo() {
		ul.header.SetText("")
		ul.hideConversations()
	}
}

func (ul *unifiedLayout) onCloseClicked() {
	page := ul.notebook.GetCurrentPage()
	ul.closeNthConversation(page)
}

func (ul *unifiedLayout) onSwitchPage(notebook gtki.Notebook, page gtki.Widget, idx int) {
	if ul.inPageSet {
		return
	}
	if csi := ul.itemMap[idx]; csi != nil {
		csi.bringToFront()
	}
	ul.cl.removeSelection()
}

func (ul *unifiedLayout) displayFirstConvo() bool {
	if iter, ok := ul.cl.model.GetIterFirst(); ok {
		if csi := ul.cl.getItemForIter(iter); csi != nil {
			csi.bringToFront()
			return true
		}
	}
	return false
}

func (ul *unifiedLayout) nextTab(gtki.Window) {
	page := ul.notebook.GetCurrentPage()
	np := (ul.notebook.GetNPages() - 1)
	if page < 0 || np < 0 {
		return
	}
	if page == np {
		ul.notebook.SetCurrentPage(0)
	} else {
		ul.notebook.NextPage()
	}
	ul.update()
}

func (ul *unifiedLayout) previousTab(gtki.Window) {
	page := ul.notebook.GetCurrentPage()
	np := (ul.notebook.GetNPages() - 1)
	if page < 0 || np < 0 {
		return
	}
	if page > 0 {
		ul.notebook.PrevPage()
	} else {
		ul.notebook.SetCurrentPage(np)
	}
	ul.update()
}

func (ul *unifiedLayout) toggleFullscreen(gtki.Window) {
	if ul.isFullscreen {
		ul.ui.window.Unfullscreen()
	} else {
		ul.ui.window.Fullscreen()
	}
	ul.isFullscreen = !ul.isFullscreen
}

func (ul *unifiedLayout) update() {
	for it, ok := ul.cl.model.GetIterFirst(); ok; ok = ul.cl.model.IterNext(it) {
		csi := ul.cl.getItemForIter(it)
		if csi != nil {
			ul.cl.updateItem(csi)
		}
	}
}
