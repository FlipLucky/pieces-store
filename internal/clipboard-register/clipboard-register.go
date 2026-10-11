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
	namedRegister   map[rune]string
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
func (c *ClipboardRegister) StoreRegisterValue(
	value string,
	registerType RegisterType,
	linewise bool,
) {
	entry := Entry{Value: value, Linewise: linewise}
	appendToRegister(entry, &c.unnamedRegister)
	if register := c.getRegister(registerType); register != nil {
		appendToRegister(entry, register)
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
