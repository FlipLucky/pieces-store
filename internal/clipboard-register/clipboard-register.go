package clipboardregister

type RegisterType int

// Entry is one stored register value together with whether it came from a
// linewise operation (yy/dd/cc — a whole line, via keyengine's generic
// doubled-verb/LineMotion mechanism) or a charwise one (yiw/diw/ciw and
// friends — an arbitrary span of text). Paste needs this to decide whether
// pasted text becomes its own fresh line or splices in at the cursor the
// way a plain text insert does.
type Entry struct {
	Value    string
	Linewise bool
}

type Register []Entry

const (
	Unnamed RegisterType = iota
	Yank
	Delete
	Visual
)

type ClipboardRegister struct {
	unnamedRegister Register
	namedRegister   map[rune]Entry
	yankRegister    Register
	visualRegister  Register
	deleteRegister  Register
}

func (c *ClipboardRegister) getRegister(
	register RegisterType,
) *Register {
	switch register {
	case Yank:
		return &c.yankRegister
	case Delete:
		return &c.deleteRegister
	case Visual:
		return &c.visualRegister
	}
	return nil
}

// StoreRegisterValue always updates the unnamed register (every delete/
// yank is pastable with a bare p/P, matching real vim) and the category
// register for registerType. named is additionally the user-selected
// register letter from a "{letter} prefix (e.g. "ayw stores into 'a' too);
// 0 means no named register was selected for this command.
func (c *ClipboardRegister) StoreRegisterValue(
	value string,
	registerType RegisterType,
	linewise bool,
	named rune,
) {
	entry := Entry{Value: value, Linewise: linewise}
	appendToRegister(entry, &c.unnamedRegister)
	if register := c.getRegister(registerType); register != nil {
		appendToRegister(entry, register)
	}
	if named != 0 {
		if c.namedRegister == nil {
			c.namedRegister = make(map[rune]Entry)
		}
		c.namedRegister[named] = entry
	}
}

func appendToRegister(
	entry Entry,
	register *Register,
) {
	*register = append(Register{entry}, (*register)...)
	if len(*register) > 9 {
		*register = (*register)[:9]
	}
}

func (c *ClipboardRegister) GetValueFromRegister(
	registerType RegisterType, index int,
) (Entry, bool) {
	register := c.getRegister(registerType)
	if register == nil || (index < 0 || index >= len(*register)) {
		return Entry{}, false
	}
	return (*register)[index], true
}

func (c *ClipboardRegister) GetValueFromUnnamedRegister(
	index int,
) (Entry, bool) {
	if index < 0 || index >= len(c.unnamedRegister) {
		return Entry{}, false
	}
	return c.unnamedRegister[index], true
}

func (c *ClipboardRegister) Paste() (Entry, bool) {
	return c.GetValueFromUnnamedRegister(0)
}

// GetValueFromNamedRegister looks up a single named register by its
// letter — single-value, no history, unlike the category registers
// (matches real vim's own a-z registers; this project deliberately
// skips the uppercase-append convention, lowercase only).
func (c *ClipboardRegister) GetValueFromNamedRegister(name rune) (Entry, bool) {
	entry, ok := c.namedRegister[name]
	return entry, ok
}

// TopEntries returns up to n most-recent entries for registerType, most
// recent first — the register overview's (":register") read path.
// Unnamed is special-cased since getRegister doesn't resolve it (it's
// always updated directly, never routed through the splitter).
func (c *ClipboardRegister) TopEntries(registerType RegisterType, n int) []Entry {
	var register *Register
	if registerType == Unnamed {
		register = &c.unnamedRegister
	} else {
		register = c.getRegister(registerType)
	}
	if register == nil {
		return nil
	}
	if n > len(*register) {
		n = len(*register)
	}
	return (*register)[:n]
}

// NamedEntries returns every currently-set named register, keyed by
// letter — the register overview's read path for the a-z registers.
func (c *ClipboardRegister) NamedEntries() map[rune]Entry {
	return c.namedRegister
}
