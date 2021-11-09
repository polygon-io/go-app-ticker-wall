package notifications

import (
	"time"

	ease "github.com/fogleman/ease"
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/polygon-io/nanovgo"
	"github.com/sirupsen/logrus"
)

type Notification struct {
	mgr          *Manager
	announcement *models.Announcement
	HasCompleted bool

	// State attributes.
	transformationAnimationOut func(float64) float64
	transformationAnimationIn  func(float64) float64
}

// setup gets the stateful attributes ready for rendering.
func (n *Notification) setup() {
	n.determineAnimations()
}

// ShouldRender checks to see if this notification should be rendered. If it's past the given lifespan
// then we set it to completed, so we can garbage collect it.
func (n *Notification) ShouldRender() bool {
	// Current timestamp ( MS )
	t := time.Now().UnixMilli()

	// We are outside of this messages lifespan, disregard.
	if t < n.announcement.ShowAtTimestampMS {
		return false
	} else if t > (n.announcement.ShowAtTimestampMS + n.announcement.LifespanMS + int64(n.mgr.settings.AnimationDurationMS)) {
		// This is past our timestamps, GC this announcement.
		n.HasCompleted = true
		return false
	}

	return true
}

// determineAnimations sets the animation effects for the intro/outro of the notification.
func (n *Notification) determineAnimations() {
	switch models.AnnouncementAnimation(n.announcement.Animation) {
	case models.AnnouncementAnimationBounce:
		// bounce looks weird on out, this seems more natural.
		n.transformationAnimationOut = ease.InElastic
		n.transformationAnimationIn = ease.OutBounce
	case models.AnnouncementAnimationEase:
		n.transformationAnimationOut = ease.InQuint
		n.transformationAnimationIn = ease.OutQuint
	case models.AnnouncementAnimationBack:
		n.transformationAnimationOut = ease.InBack
		n.transformationAnimationIn = ease.OutBack
	default:
		n.transformationAnimationOut = ease.InElastic
		n.transformationAnimationIn = ease.OutElastic
	}
}

// Render actually renders the notification to the GUI context. `ShouldRender` should be run before this
// to ensure the rendering method should be called on this notification.
func (n *Notification) Render(ctx *nanovgo.Context) {
	// Current timestamp ( MS )
	t := time.Now().UnixMilli()

	// Get necessary parameters.
	settings := n.mgr.settings
	screen := n.mgr.screen
	cluster := n.mgr.cluster

	// Text Settings.
	textTopStart := float64(-300)
	textTopEnd := float64(140)
	textTop := textTopEnd

	// BG Settings.
	bgBottomStart := float64(0)
	bgBottomEnd := float64(screen.Height)
	bgBottom := bgBottomEnd
	bgTop := (bgBottom - float64(screen.Height))

	// Determine which animation to use for the announcement.
	// To see more: https://github.com/fogleman/ease

	if t-n.announcement.ShowAtTimestampMS < int64(settings.AnimationDurationMS) {
		// Enter animation is in progress.
		diff := t - n.announcement.ShowAtTimestampMS
		percentageCompleted := float64(diff) / float64(settings.AnimationDurationMS)
		logrus.Info("Enter completion: ", percentageCompleted)

		// bg calcs
		inPercCompleted := n.transformationAnimationIn(percentageCompleted)
		bgBottom = bgBottomStart - ((bgBottomStart - bgBottomEnd) * inPercCompleted)
		bgTop = (bgBottom - float64(screen.Height))

		// text calcs
		textTop = textTopStart - ((textTopStart - textTopEnd) * inPercCompleted)

	} else if t > n.announcement.ShowAtTimestampMS+n.announcement.LifespanMS {
		// Exit animation in progress.
		diff := t - (n.announcement.ShowAtTimestampMS + n.announcement.LifespanMS)
		percentageCompleted := float64(diff) / float64(settings.AnimationDurationMS)
		logrus.Info("Exit completion: ", percentageCompleted)

		// bg calcs
		outPercCompleted := n.transformationAnimationOut(percentageCompleted)
		bgBottom = bgBottomEnd - ((bgBottomEnd - bgBottomStart) * outPercCompleted)
		bgTop = (bgBottom - float64(screen.Height))

		// text calcs
		textTop = textTopEnd - ((textTopEnd - textTopStart) * outPercCompleted)
	}

	ctx.Save()
	defer ctx.Restore()

	ctx.BeginPath()
	// Determine where the box should start ( may not be on our screen ).
	screenGlobalOffset := cluster.ScreenGlobalOffset(screen.UUID)
	left := -float32(screenGlobalOffset)
	// Position bg.
	ctx.RoundedRect(left, float32(bgTop), float32(cluster.GlobalViewportSize()), float32(bgBottom), 0)

	// Determine background color based on announcement type:].
	if n.announcement.AnnouncementType == int32(models.AnnouncementTypeDanger) {
		ctx.SetFillColor(nanovgo.RGBA(255, 122, 122, 222))
	} else if n.announcement.AnnouncementType == int32(models.AnnouncementTypeSuccess) {
		ctx.SetFillColor(nanovgo.RGBA(122, 255, 122, 222))
	} else {
		ctx.SetFillColor(nanovgo.RGBA(122, 122, 255, 222))
	}

	ctx.Fill()

	ctx.SetFontSize(96.0)
	ctx.SetFontFace("sans-bold")
	ctx.SetTextAlign(nanovgo.AlignCenter | nanovgo.AlignMiddle)

	// ctx.SetFontBlur(0)
	ctx.SetFillColor(nanovgo.RGBA(255, 255, 255, 255))
	middle := (float32(cluster.GlobalViewportSize()) / 2) - float32(screenGlobalOffset)
	ctx.Text(middle, float32(textTop), n.announcement.Message)
}
