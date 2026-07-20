package testutil

import tea "charm.land/bubbletea/v2"

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
