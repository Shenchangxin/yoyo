package desktop

import (
	"errors"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/winsize"
)

// companionOrigin 把宠物放在主工作区右下角，右边距 28，底边距是 companionBottomClearance，再夹进屏幕。
func companionOrigin() (x, y int) {
	aw, ah := primaryWorkArea()
	x = aw - companionW - 28
	y = ah - companionIdleH - companionBottomClearance
	vx, vy, vw, vh := virtualScreen()
	return clampCompanionTo(x, y, companionW, companionIdleH, companionBottomClearance, companionWorkBoxes(), vx, vy, vw, vh)
}

// companionWorkBoxes 收集每块屏幕的工作区。工作区没有尺寸时用屏幕边界。取不到应用时返回 nil。
func companionWorkBoxes() []hitBox {
	gui := application.Get()
	if gui == nil {
		return nil
	}
	var out []hitBox
	for _, scr := range gui.Screen.GetAll() {
		if scr == nil {
			continue
		}
		work := scr.WorkArea
		if work.Width <= 0 || work.Height <= 0 {
			work = scr.Bounds
		}
		if work.Width <= 0 || work.Height <= 0 {
			continue
		}
		out = append(out, hitBox{work.X, work.Y, work.Width, work.Height})
	}
	return out
}

// clampCompanion 按当前形态取宽高：通话中用通话卡片，否则用宠物宽和字幕框高度。字幕打开时底边距用 companionBottomClearance，否则用 companionPad。然后夹进最近的工作区和虚拟屏幕。
// x 传给 clampCompanionTo。
// y 传给 clampCompanionTo。
func clampCompanion(x, y int) (int, int) {
	w, h := companionW, companionFrameH(companionCaptionOn.Load())
	pad := companionPad
	if companionInteractive.Load() {
		w, h = companionCallW, companionCallH
	} else if companionCaptionOn.Load() {
		pad = companionBottomClearance
	}
	vx, vy, vw, vh := virtualScreen()
	return clampCompanionTo(x, y, w, h, pad, companionWorkBoxes(), vx, vy, vw, vh)
}

// companionCursorLocal 优先用系统原生坐标。原生取不到时，用 CSS 光标减去窗口位置。窗口为空或没有光标时返回假。
// win 传给 companionCursorLocalNative。原生取不到且 win 非 nil 时，用它的 Position 和 Size 换算 CSS 光标。
func companionCursorLocal(win application.Window) (lx, ly, ww, wh float64, ok bool) {
	if lx, ly, ww, wh, ok = companionCursorLocalNative(win); ok {
		return lx, ly, ww, wh, true
	}
	cx, cy, ok := desktopCursorCSS()
	if !ok || win == nil {
		return 0, 0, 0, 0, false
	}
	wx, wy := win.Position()
	wi, hi := win.Size()
	return cx - float64(wx), cy - float64(wy), float64(wi), float64(hi), true
}

// SetCompanionCaption 写下字幕开关。和上次相同则返回。变化后 syncCompanionFrame。
func (s *Service) SetCompanionCaption(on bool) {
	companionCaptionOn.Store(on)
	if s.companionCaption.Swap(on) == on {
		return
	}
	s.syncCompanionFrame()
}

// SetCompanionCaptionSize 更新字幕盒子。宽高都没变则返回。字幕正开着时再 syncCompanionFrame。
// w 传给 setCompanionCaptionBox。
// h 传给 setCompanionCaptionBox。
func (s *Service) SetCompanionCaptionSize(w, h int) {
	beforeW := companionCaptionW.Load()
	beforeH := companionCaptionH.Load()
	setCompanionCaptionBox(w, h)
	if companionCaptionW.Load() == beforeW && companionCaptionH.Load() == beforeH {
		return
	}
	if s.companionCaption.Load() {
		s.syncCompanionFrame()
	}
}

// syncCompanionFrame 在窗口为空时返回。通话中只重贴窗口区域。空闲时把窗口改成宠物宽和字幕框高度，避开隐藏哨兵后夹住位置，再贴区域。
func (s *Service) syncCompanionFrame() {
	win := s.companion
	if win == nil || s.companionCall.Load() {
		if win != nil {
			applyCompanionShape(win, s.companionCaption.Load() || s.companionCall.Load())
		}
		return
	}
	on := s.companionCaption.Load()
	companionCaptionOn.Store(on)
	wantH := companionFrameH(on)
	x, y := win.Position()
	if winsize.HiddenSentinel(x, y) {
		applyCompanionShape(win, on)
		return
	}
	w, h := win.Size()
	nx, ny := clampCompanion(x, y)
	if nx != x || ny != y {
		if s.companionClamping.CompareAndSwap(false, true) {
			win.SetPosition(nx, ny)
			s.companionClamping.Store(false)
		}
	}
	if w != companionW || h != wantH {
		win.SetSize(companionW, wantH)
	}
	applyCompanionShape(win, on)
}

// PlaceCompanion 把宠物窗口移到夹过的坐标。窗口为空、坐标是隐藏哨兵，或位置没变时返回。companionClamping 正在进行时也不再套一层。
// x 先交给 winsize.HiddenSentinel，再传给 clampCompanion。
// y 先交给 winsize.HiddenSentinel，再传给 clampCompanion。
func (s *Service) PlaceCompanion(x, y int) {
	win := s.companion
	if win == nil {
		return
	}
	if winsize.HiddenSentinel(x, y) {
		return
	}
	nx, ny := clampCompanion(x, y)
	wx, wy := win.Position()
	if nx == wx && ny == wy {
		return
	}
	if !s.companionClamping.CompareAndSwap(false, true) {
		return
	}
	defer s.companionClamping.Store(false)
	win.SetPosition(nx, ny)
}

// pinCompanion 在窗口被拖动或系统改过位置后，把坐标夹回工作区。隐藏哨兵不改。
// win 为 nil 时返回。否则读取 Position，夹过之后必要时 SetPosition。
func (s *Service) pinCompanion(win application.Window) {
	if win == nil || !s.companionClamping.CompareAndSwap(false, true) {
		return
	}
	defer s.companionClamping.Store(false)
	x, y := win.Position()
	if winsize.HiddenSentinel(x, y) {
		return
	}
	nx, ny := clampCompanion(x, y)
	if nx != x || ny != y {
		win.SetPosition(nx, ny)
	}
}

// companionShown 在宠物窗口还在且没有被收起时返回真。
func (s *Service) companionShown() bool {
	return s.companion != nil && !s.companionDismissed.Load()
}

// companionIgnoresHits 在专注模式开着且不在通话中时返回真，点击会穿透。
func (s *Service) companionIgnoresHits() bool {
	return s.companionFocus.Load() && !s.companionCall.Load()
}

// SetCompanionFocus 在开关变化时同步点击穿透，并 refreshChrome。
func (s *Service) SetCompanionFocus(on bool) {
	if s.companionFocus.Swap(on) == on {
		return
	}
	s.syncCompanionClickThrough()
	s.refreshChrome()
}

// syncCompanionClickThrough 按 companionIgnoresHits 记下穿透标志并重贴窗口区域。需要穿透时调用 setCompanionPassthrough。
func (s *Service) syncCompanionClickThrough() {
	through := s.companionIgnoresHits()
	companionClickThrough.Store(through)
	win := s.companion
	if win == nil {
		return
	}
	applyCompanionShape(win, s.companionCaption.Load() || s.companionCall.Load())
	if through {
		setCompanionPassthrough(win, true)
	}
}

// ResetCompanion 把宠物窗口放回 companionOrigin。没有窗口时返回。
func (s *Service) ResetCompanion() {
	win := s.companion
	if win == nil {
		return
	}
	nx, ny := companionOrigin()
	win.SetPosition(nx, ny)
}

// beginCompanionCall 把通话标成真。已经在通话中时只保持 interactive，不再发事件。第一次进入时套用通话几何，发出 yoyo:call 且值为真，并 refreshChrome。
func (s *Service) beginCompanionCall() {
	if s.companionCall.Swap(true) {
		companionInteractive.Store(true)
		return
	}
	companionInteractive.Store(true)
	s.applyCompanionCallGeometry()
	if s.gui != nil {
		s.gui.Event.Emit("yoyo:call", true)
	}
	s.refreshChrome()
}

// endCompanionCall 把通话标成假。原本就不在通话中则返回。resizeAndRefresh 为真时恢复记下的位置、同步外框和穿透，并 settleCompanionShape。只要曾经在通话中，就发出 yoyo:call 且值为假。resizeAndRefresh 为真时再 refreshChrome。
func (s *Service) endCompanionCall(resizeAndRefresh bool) {
	was := s.companionCall.Swap(false)
	companionInteractive.Store(false)
	if !was {
		return
	}
	if resizeAndRefresh {
		if win := s.companion; win != nil {
			s.restoreCompanionHome(win)
			s.syncCompanionFrame()
			s.syncCompanionClickThrough()
			s.settleCompanionShape()
		}
	}
	if s.gui != nil {
		s.gui.Event.Emit("yoyo:call", false)
	}
	if resizeAndRefresh {
		s.refreshChrome()
	}
}

// companionCallOrigin 保持空闲窗口的右下角不动，通话窗口向左上长大。宽或高小于 32 时按空闲尺寸算。
// x 加上夹过的宽，再减去 companionCallW。
// y 加上夹过的高，再减去 companionCallH。
// w 小于 32 时改成 companionW。
// h 小于 32 时改成 companionIdleH。
func companionCallOrigin(x, y, w, h int) (int, int) {
	if w < 32 {
		w = companionW
	}
	if h < 32 {
		h = companionIdleH
	}
	return x + w - companionCallW, y + h - companionCallH
}

// rememberCompanionHome 只记第一次，而且窗口还没大过宠物尺寸加 24 时才记。已经是通话尺寸时不覆盖回家坐标。
// x 在尚未记下且宽高都没有超过宠物尺寸加 24 时写入 companionHomeX。
// y 在同一条件下写入 companionHomeY。
// w 大于 companionW 加 24 时直接返回。
// h 大于 companionH 加 24 时直接返回。
func (s *Service) rememberCompanionHome(x, y, w, h int) {
	if s.companionHomeOK.Load() {
		return
	}
	if w > companionW+24 || h > companionH+24 {
		return
	}
	s.companionHomeX.Store(int32(x))
	s.companionHomeY.Store(int32(y))
	s.companionHomeOK.Store(true)
}

// restoreCompanionHome 把窗口缩回字幕框高度，放回记下的坐标，清掉标记，并按字幕状态贴区域。
// win 为 nil 或还没记下回家坐标时返回。否则 SetSize、SetPosition，并传给 applyCompanionShape。
func (s *Service) restoreCompanionHome(win application.Window) {
	if win == nil || !s.companionHomeOK.Load() {
		return
	}
	x, y := int(s.companionHomeX.Load()), int(s.companionHomeY.Load())
	s.companionHomeOK.Store(false)
	win.SetSize(companionW, companionFrameH(s.companionCaption.Load()))
	win.SetPosition(x, y)
	applyCompanionShape(win, s.companionCaption.Load())
}

// applyCompanionCallGeometry 在通话中记住回家坐标，按右下角对齐放大到通话尺寸；窗口已经比宠物大 24 以上时不再改原点。然后关掉穿透并按字幕贴区域。通话关闭时同步外框和穿透；若当前不是专注穿透，再把 setCompanionPassthrough 设为真。
func (s *Service) applyCompanionCallGeometry() {
	win := s.companion
	on := s.companionCall.Load()
	companionInteractive.Store(on)
	if win == nil {
		return
	}
	defer s.settleCompanionShape()
	if on {
		x, y := win.Position()
		w, h := win.Size()
		s.rememberCompanionHome(x, y, w, h)
		nx, ny := companionCallOrigin(x, y, w, h)
		if w > companionW+24 || h > companionH+24 {
			nx, ny = x, y
		}
		nx, ny = clampCompanion(nx, ny)
		win.SetSize(companionCallW, companionCallH)
		if nx != x || ny != y {
			win.SetPosition(nx, ny)
		}
		s.syncCompanionClickThrough()
		setCompanionPassthrough(win, false)
		applyCompanionShape(win, true)
		return
	}
	s.syncCompanionFrame()
	s.syncCompanionClickThrough()
	if !s.companionIgnoresHits() {
		setCompanionPassthrough(win, true)
	}
}

// settleCompanionShape 在 30、50、80、120、200、300 毫秒后再贴一次窗口区域，等 WebView 子窗口赶上通话尺寸。世代变了或正在退出则停下。
func (s *Service) settleCompanionShape() {
	if s == nil || s.companion == nil {
		return
	}
	gen := s.companionShapeGen.Add(1)
	go func() {
		for _, wait := range []time.Duration{
			30 * time.Millisecond,
			50 * time.Millisecond,
			80 * time.Millisecond,
			120 * time.Millisecond,
			200 * time.Millisecond,
			300 * time.Millisecond,
		} {
			time.Sleep(wait)
			if s.companionShapeGen.Load() != gen || s.quitting.Load() {
				return
			}
			win := s.companion
			if win == nil {
				return
			}
			applyCompanionShape(win, s.companionCaption.Load() || s.companionCall.Load())
		}
	}()
}

// replayCompanionCall 再发一次 yoyo:call，值是当前是否在通话。gui 为空时返回。
func (s *Service) replayCompanionCall() {
	if s.gui == nil {
		return
	}
	s.gui.Event.Emit("yoyo:call", s.companionCall.Load())
}

// OpenCompanionChat 优先 RaiseSession 最近一次脉冲的会话，否则取第一个正在跑的会话。都没有则 RaiseWindow。最后 refreshChrome。
func (s *Service) OpenCompanionChat() {
	if sid, _ := s.lastPulseSession.Load().(string); strings.TrimSpace(sid) != "" {
		s.RaiseSession(sid)
		s.refreshChrome()
		return
	}
	if s.App != nil || s.RPC != nil {
		if ids := s.RunningIDs(); len(ids) > 0 {
			s.RaiseSession(ids[0])
			s.refreshChrome()
			return
		}
	}
	s.RaiseWindow()
	s.refreshChrome()
}

// ToggleCompanion 在宠物正显示时 HideCompanion，否则 ShowCompanion。
func (s *Service) ToggleCompanion() {
	if s.companionShown() {
		s.HideCompanion()
		return
	}
	_ = s.ShowCompanion()
}

// syncCompanion 在配置关掉宠物时收起、关闭窗口，并发出 yoyo:companion 且值为 hidden。已经收起时只关闭并 refreshChrome。否则 OpenCompanion。
// cfg 的 CompanionEnabled 为假时收起并关闭宠物。为真且尚未收起时调用 OpenCompanion。
func (s *Service) syncCompanion(cfg app.Config) {
	if !cfg.CompanionEnabled {
		s.companionDismissed.Store(true)
		s.CloseCompanion()
		s.emitCompanion("hidden")
		return
	}
	if s.companionDismissed.Load() {
		s.CloseCompanion()
		s.refreshChrome()
		return
	}
	_ = s.OpenCompanion()
}

// PrepareCompanion 按当前配置调用 syncCompanion。
func (s *Service) PrepareCompanion() {
	s.syncCompanion(s.GetConfig())
}

// CloseCompanion 以不改尺寸的方式结束通话，清空窗口字段，并在 companionClosing 为真时关掉窗口，避免关闭钩子把它当成用户收起。
func (s *Service) CloseCompanion() {
	s.endCompanionCall(false)
	win := s.companion
	s.companion = nil
	if win != nil {
		s.companionClosing.Store(true)
		win.Close()
		s.companionClosing.Store(false)
	}
}

// DismissCompanion 把宠物标成已收起，关闭窗口，并发出 yoyo:companion 且值为 hidden。
func (s *Service) DismissCompanion() {
	s.companionDismissed.Store(true)
	s.CloseCompanion()
	s.emitCompanion("hidden")
}

// HideCompanion 调用 DismissCompanion。
func (s *Service) HideCompanion() {
	s.DismissCompanion()
}

// ShowCompanion 清掉收起标记。配置里宠物没开时返回 nil。否则 OpenCompanion。
func (s *Service) ShowCompanion() error {
	s.companionDismissed.Store(false)
	if !s.GetConfig().CompanionEnabled {
		return nil
	}
	return s.OpenCompanion()
}

// emitCompanion 在 gui 非空时发出 yoyo:companion，载荷是 action，然后 refreshChrome。现有动作是 shown 和 hidden。
func (s *Service) emitCompanion(action string) {
	if s.gui != nil {
		s.gui.Event.Emit("yoyo:companion", action)
	}
	s.refreshChrome()
}

// OpenCompanion 在 gui 为空时返回 no window host。窗口已在时显示、置顶、藏起任务栏按钮、套几何、夹位置、重放通话状态，并发出 yoyo:companion 且值为 shown。
// 否则新建名为 yoyo-companion 的无边框透明置顶窗口，地址是 /?surface=companion，空闲尺寸，最大到通话尺寸。非退出、非内部关闭时，关闭钩子会收起并发出 hidden。移动结束时夹回工作区。运行时就绪后若配置仍允许且未收起，才显示。
func (s *Service) OpenCompanion() error {
	if s.gui == nil {
		return errors.New("no window host")
	}
	s.companionDismissed.Store(false)
	s.startCursorWatch()
	if s.companion != nil {
		s.companion.Show()
		s.companion.SetAlwaysOnTop(true)
		hideCompanionTaskbar(s.companion)
		s.applyCompanionCallGeometry()
		s.pinCompanion(s.companion)
		s.replayCompanionCall()
		s.emitCompanion("shown")
		return nil
	}
	x, y := companionOrigin()
	opts := application.WebviewWindowOptions{
		Name:                  companionWindowName,
		Title:                 "Yoyo",
		Width:                 companionW,
		Height:                companionIdleH,
		MinWidth:              companionW,
		MinHeight:             companionIdleH,
		MaxWidth:              companionCallW,
		MaxHeight:             companionCallH,
		DisableResize:         true,
		Frameless:             true,
		AlwaysOnTop:           true,
		BackgroundType:        application.BackgroundTypeTransparent,
		BackgroundColour:      application.NewRGBA(0, 0, 0, 0),
		URL:                   "/?surface=companion",
		InitialPosition:       application.WindowXY,
		X:                     x,
		Y:                     y,
		MinimiseButtonState:   application.ButtonHidden,
		MaximiseButtonState:   application.ButtonHidden,
		CloseButtonState:      application.ButtonHidden,
		FullscreenButtonState: application.ButtonHidden,
		Mac: application.MacWindow{
			TitleBar:    application.MacTitleBarHiddenInset,
			WindowClass: application.MacWindowClassPanel,
			PanelPreferences: application.MacPanelPreferences{
				FloatingPanel: true,
				UtilityWindow: true,
			},
		},
		Windows: application.WindowsWindow{
			Theme:                             application.SystemDefault,
			DisableMenu:                       true,
			DisableFramelessWindowDecorations: true,
			NonClientRegionSupport:            true,
			HiddenOnTaskbar:                   true,
			WindowDidMoveDebounceMS:           16,
		},
		Linux: application.LinuxWindow{
			Icon: AppIcon,
		},
	}
	win := s.gui.Window.NewWithOptions(opts)
	s.companion = win
	s.companionPlaced.Store(false)
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if s.companion == win {
			s.companion = nil
		}
		if s.Quitting() || s.companionClosing.Load() {
			s.endCompanionCall(false)
			return
		}
		s.companionDismissed.Store(true)
		s.endCompanionCall(false)
		s.emitCompanion("hidden")
	})
	win.OnWindowEvent(events.Common.WindowDidMove, func(*application.WindowEvent) {
		s.pinCompanion(win)
	})
	if runtime.GOOS == "windows" {
		win.OnWindowEvent(events.Windows.WindowEndMove, func(*application.WindowEvent) {
			s.pinCompanion(win)
		})
	}
	win.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		if s.companionPlaced.CompareAndSwap(false, true) {
			px, py := companionOrigin()
			s.applyCompanionCallGeometry()
			win.SetPosition(px, py)
			win.SetAlwaysOnTop(true)
		}
		if !s.GetConfig().CompanionEnabled || s.companionDismissed.Load() {
			s.CloseCompanion()
			s.refreshChrome()
			return
		}
		win.Show()
		win.SetAlwaysOnTop(true)
		hideCompanionTaskbar(win)
		s.applyCompanionCallGeometry()
		s.pinCompanion(win)
		s.replayCompanionCall()
		s.emitCompanion("shown")
	})
	return nil
}

// startCursorWatch 只启动一次，每 8 毫秒看光标。命中宠物或通话卡片时可以拖，移动超过 4 像素才算拖动，并夹在工作区内。空闲时每 4 拍夹一次位置。
// Windows 上字幕或尺寸变化时重贴区域。其他系统按是否该穿透切换 setCompanionPassthrough。光标移动超过 0.25 时执行 window.__yoyoCursor，参数是 CSS 坐标和窗口位置。
func (s *Service) startCursorWatch() {
	if !s.cursorWatch.CompareAndSwap(false, true) {
		return
	}
	go func() {
		tick := time.NewTicker(8 * time.Millisecond)
		defer tick.Stop()
		var lastX, lastY float64
		var grabX, grabY float64
		var armed, dragging, downPrev bool
		passthrough := true
		var idlePins uint8
		var lastCap bool
		var lastWW, lastWH float64
		for range tick.C {
			if s.quitting.Load() {
				return
			}
			win := s.companion
			if win == nil || !win.IsVisible() {
				armed, dragging, downPrev = false, false, false
				lastWW, lastWH = 0, 0
				continue
			}
			caption := s.companionCaption.Load()
			lx, ly, ww, wh, locOK := companionCursorLocal(win)
			hit := locOK && CompanionHit(lx, ly, ww, wh, caption)
			down := leftMouseDown()
			ignore := s.companionIgnoresHits()
			inCall := s.companionCall.Load()
			want := true
			if locOK && !ignore {
				if inCall {
					want = false
				} else {
					want = !(hit || dragging || (armed && down))
				}
			}
			if runtime.GOOS == "windows" {
				if locOK && (caption != lastCap || absF(ww-lastWW) > 1 || absF(wh-lastWH) > 1) {
					lastCap, lastWW, lastWH = caption, ww, wh
					applyCompanionShape(win, caption || s.companionCall.Load())
				}
			} else if want != passthrough {
				passthrough = want
				setCompanionPassthrough(win, want)
			}
			cx, cy, curOK := desktopCursorCSS()
			if ignore {
				armed, dragging = false, false
			}
			if down && !downPrev && hit && curOK && !ignore {
				wx, wy := win.Position()
				armed = true
				grabX = cx - float64(wx)
				grabY = cy - float64(wy)
			}
			if !down {
				armed, dragging = false, false
			} else if armed && curOK {
				wx, wy := win.Position()
				nx := cx - grabX
				ny := cy - grabY
				if !dragging && (absF(nx-float64(wx)) >= 4 || absF(ny-float64(wy)) >= 4) {
					dragging = true
				}
				if dragging {
					ix, iy := clampCompanion(int(nx+0.5), int(ny+0.5))
					if ix != wx || iy != wy {
						win.SetPosition(ix, iy)
					}
				}
			}
			downPrev = down
			if !dragging {
				idlePins++
				if armed || idlePins >= 4 {
					idlePins = 0
					s.pinCompanion(win)
				}
			} else {
				idlePins = 0
			}
			if !curOK {
				continue
			}
			if absF(cx-lastX) < 0.25 && absF(cy-lastY) < 0.25 {
				continue
			}
			lastX, lastY = cx, cy
			wx, wy := win.Position()
			win.ExecJS("window.__yoyoCursor&&window.__yoyoCursor(" +
				strconv.FormatFloat(cx, 'f', 1, 64) + "," +
				strconv.FormatFloat(cy, 'f', 1, 64) + "," +
				strconv.Itoa(wx) + "," +
				strconv.Itoa(wy) + ")")
		}
	}()
}

// absF 返回浮点数的绝对值。
// v 小于 0 时返回相反数，否则原样返回。
func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
