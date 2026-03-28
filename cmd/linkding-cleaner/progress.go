package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

const progressPadding = 2

// progressMsg is sent by each check goroutine when it finishes.
type progressMsg struct{}

type progressModel struct {
	total     int
	completed int
	bar       progress.Model
	start     time.Time
	elapsed   time.Duration
}

func newProgressModel(total int) progressModel {
	return progressModel{
		total: total,
		bar:   progress.New(progress.WithDefaultGradient()),
		start: time.Now(),
	}
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m progressModel) Init() tea.Cmd {
	return tickCmd()
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		counterWidth := len(fmt.Sprintf("  %d/%d", m.total, m.total))
		m.bar.Width = msg.Width - progressPadding - counterWidth
		return m, nil

	case progressMsg:
		m.completed++
		cmd := m.bar.SetPercent(float64(m.completed) / float64(m.total))
		if m.completed >= m.total {
			return m, tea.Sequence(cmd, tea.Quit)
		}
		return m, cmd

	case tickMsg:
		m.elapsed = time.Since(m.start)
		return m, tickCmd()

	case progress.FrameMsg:
		updated, cmd := m.bar.Update(msg)
		m.bar = updated.(progress.Model)
		return m, cmd
	}
	return m, nil
}

func (m progressModel) View() string {
	pad := strings.Repeat(" ", progressPadding)
	return "\n" +
		pad + m.bar.View() + fmt.Sprintf("  %d/%d", m.completed, m.total) + "\n" +
		pad + "Elapsed: " + m.elapsed.Round(10*time.Millisecond).String() + "\n"
}
