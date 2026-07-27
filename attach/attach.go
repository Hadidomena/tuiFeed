package attach

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/bsk"
	"github.com/Hadidomena/tuiFeed/utils"
)

type RenderedMsg struct {
	ImageRows int
	Status    string
}

type ErrorMsg string

func Render(embeds []string, imgCursor int, yOffset int) tea.Msg {
	if imgCursor >= len(embeds) || imgCursor < 0 {
		return ErrorMsg("No image to render")
	}

	bsk.ClearImages()
	rows, err := bsk.RenderImage(embeds[imgCursor], yOffset)
	if err != nil {
		return ErrorMsg(fmt.Sprintf("Render failed: %v", err))
	}
	return RenderedMsg{
		ImageRows: rows,
		Status:    fmt.Sprintf("Image %d/%d  [←/→] navigate  [o] open externally", imgCursor+1, len(embeds)),
	}
}

func Open(embeds []string, imgCursor int) tea.Msg {
	if imgCursor >= len(embeds) || imgCursor < 0 {
		return ErrorMsg("No image to open")
	}

	data, err := utils.DownloadURL(embeds[imgCursor])
	if err != nil {
		return ErrorMsg(fmt.Sprintf("Download failed: %v", err))
	}

	if err := bsk.RenderImageExternal(data); err != nil {
		return ErrorMsg(fmt.Sprintf("Open failed: %v", err))
	}
	return ErrorMsg("Opened externally")
}
