package testutil

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/config"
)

func KeyPress(text string, code rune, mods ...tea.KeyMod) tea.KeyPressMsg {
	var mod tea.KeyMod
	for _, m := range mods {
		mod |= m
	}
	return tea.KeyPressMsg(tea.Key{Text: text, Code: code, Mod: mod})
}

func KeyRune(r rune) tea.KeyPressMsg {
	return KeyPress(string(r), r)
}

func KeySpecial(code rune) tea.KeyPressMsg {
	return KeyPress("", code)
}

func SetupTestConfig(t *testing.T) *config.Config {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	cfg := &config.Config{}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func UpdateModel[M tea.Model](m M, msg tea.Msg) (M, *tea.Cmd) {
	newModel, cmd := m.Update(msg)
	result := newModel.(M)
	if cmd != nil {
		return result, &cmd
	}
	return result, nil
}
