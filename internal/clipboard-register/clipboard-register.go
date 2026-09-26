package clipboardregister

type RegisterType int
type Register []string

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
) {
	appendToRegister(value, &c.unnamedRegister)
	if register := c.getRegister(registerType); register != nil {
		appendToRegister(value, register)
	}

}

func appendToRegister(
	value string,
	register *Register,
) {
	*register = append(Register{value}, (*register)...)
	if len(*register) > 9 {
		*register = (*register)[:9]
	}
}

func (c *ClipboardRegister) GetValueFromRegister(
	registerType RegisterType, index int,
) (string, bool) {
	register := c.getRegister(registerType)
	if register == nil || (index < 0 || index >= len(*register)) {
		return "", false
	}
	return (*register)[index], true
}

func (c *ClipboardRegister) GetValueFromUnnamedRegister(
	index int,
) (string, bool) {
	if index < 0 || index >= len(c.unnamedRegister) {
		return "", false
	}
	return c.unnamedRegister[index], true
}

func (c *ClipboardRegister) Paste() (string, bool) {
	return c.GetValueFromUnnamedRegister(0)
}
