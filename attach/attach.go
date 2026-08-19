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

const footerReserve = 4

func Render(embeds []string, imgCursor int, yOffset int, maxCols int, maxRows int) tea.Msg {
	if imgCursor >= len(embeds) || imgCursor < 0 {
		return ErrorMsg("No image to render")
	}
	if maxRows <= 0 || yOffset < 0 {
		return ErrorMsg("Not enough room to render image")
	}

	bsk.ClearImages()
	rows, err := bsk.RenderImage(embeds[imgCursor], yOffset, maxCols, maxRows)
	if err != nil {
		return ErrorMsg(fmt.Sprintf("Render failed: %v", err))
	}
	return RenderedMsg{
		ImageRows: rows,
		Status:    fmt.Sprintf("Image %d/%d  [←/→] navigate  [o] open externally", imgCursor+1, len(embeds)),
	}
}

func ComputeMaxRows(termHeight int, yOffset int) int {
	available := termHeight - yOffset - footerReserve
	if available < 3 {
		return 0
	}
	return available
}

func ComputeMaxCols(termWidth int) int {
	cw := utils.ContentWidth(termWidth)
	if cw <= 0 {
		return 0
	}
	maxCols := cw * 2 / 3
	if maxCols > 60 {
		maxCols = 60
	}
	if maxCols < 20 {
		maxCols = 20
	}
	if maxCols > cw {
		maxCols = cw
	}
	return maxCols
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
