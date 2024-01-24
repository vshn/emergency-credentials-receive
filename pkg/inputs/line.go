package inputs

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var ErrInterrupted = fmt.Errorf("interrupted")

func PassphraseInput(header, placeholder string) (string, error) {
	p := tea.NewProgram(newLineModel(header, placeholder, true))

	m, err := p.Run()
	if err != nil {
		return "", err
	}

	return m.(lineModel).textinput.Value(), m.(lineModel).err
}

func LineInput(header, placeholder string) (string, error) {
	p := tea.NewProgram(newLineModel(header, placeholder, false))

	m, err := p.Run()
	if err != nil {
		return "", err
	}

	return m.(lineModel).textinput.Value(), m.(lineModel).err
}

type lineModel struct {
	header string

	textinput textinput.Model
	err       error
}

func newLineModel(header, placeholder string, pw bool) lineModel {
	ti := textinput.New()
	if pw {
		ti.EchoMode = textinput.EchoPassword
	}
	ti.Placeholder = placeholder
	ti.Focus()

	return lineModel{
		header:    header,
		textinput: ti,
		err:       nil,
	}
}

func (m lineModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m lineModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlD, tea.KeyEnter:
			return m, tea.Quit
		case tea.KeyCtrlC:
			m.err = ErrInterrupted
			return m, tea.Quit
		default:
			if !m.textinput.Focused() {
				cmds = append(cmds, m.textinput.Focus())
			}
		}
	}

	var cmd tea.Cmd
	m.textinput, cmd = m.textinput.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m lineModel) View() string {
	return fmt.Sprintf(
		"\n%s\n\n%s\n\n%s\n\n",
		m.header,
		m.textinput.View(),
		"(ctrl+c to quit)",
	)
}
