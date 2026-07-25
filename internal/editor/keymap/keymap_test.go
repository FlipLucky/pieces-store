package keymap

import (
	"testing"
)

type MockEditor struct {
	Mode         int
	LeftCalls    int
	RightCalls   int
	UpCalls      int
	DownCalls    int
	DeleteCalls  int
	UndoCalls    int
	InsertCalls  int
	CmdBuffer    string
	ExecutedCmd  string
}

func (m *MockEditor) MoveCursorLeft()  { m.LeftCalls++ }
func (m *MockEditor) MoveCursorRight() { m.RightCalls++ }
func (m *MockEditor) MoveCursorUp()    { m.UpCalls++ }
func (m *MockEditor) MoveCursorDown()  { m.DownCalls++ }
func (m *MockEditor) InsertText(data []byte) { m.InsertCalls++ }
func (m *MockEditor) DeleteText()      { m.DeleteCalls++ }
func (m *MockEditor) Undo() bool       { m.UndoCalls++; return true }
func (m *MockEditor) SetModeInt(mode int) { m.Mode = mode }
func (m *MockEditor) GetModeInt() int  { return m.Mode }
func (m *MockEditor) AppendCommandBuffer(r rune) { m.CmdBuffer += string(r) }
func (m *MockEditor) BackspaceCommandBuffer() {
	if len(m.CmdBuffer) > 0 {
		m.CmdBuffer = m.CmdBuffer[:len(m.CmdBuffer)-1]
	}
}
func (m *MockEditor) ClearCommandBuffer() { m.CmdBuffer = "" }
func (m *MockEditor) GetCommandBuffer() string { return m.CmdBuffer }
func (m *MockEditor) ExecuteCommand(cmd string) error {
	m.ExecutedCmd = cmd
	m.SetModeInt(0)
	return nil
}

func TestKeymapRouter(t *testing.T) {
	router := NewRouter()
	graphs := BuildDefaultGraphs()
	for mode, graph := range graphs {
		router.RegisterModeGraph(mode, graph)
	}

	mock := &MockEditor{Mode: 0}

	// 1. Single motion: 'j'
	handled := router.HandleKey(mock, "j")
	if !handled || mock.DownCalls != 1 {
		t.Errorf("Expected 1 Down call, got %d", mock.DownCalls)
	}

	// 2. Counted motion: '3j'
	router.HandleKey(mock, "3")
	router.HandleKey(mock, "j")
	if mock.DownCalls != 4 { // 1 + 3 = 4
		t.Errorf("Expected 4 Down calls total, got %d", mock.DownCalls)
	}

	// 3. Multi-character verb sequence: 'd' -> 'd'
	router.HandleKey(mock, "d")
	if router.currentNode == nil || router.currentNode.Name != "d" {
		t.Errorf("Expected intermediate node 'd', got %+v", router.currentNode)
	}
	router.HandleKey(mock, "d")
	if mock.DeleteCalls != 1 {
		t.Errorf("Expected 1 Delete call from 'dd', got %d", mock.DeleteCalls)
	}

	// 4. Modifier verb sequence: 'd' -> 'i' -> 'w'
	mock.DeleteCalls = 0
	router.HandleKey(mock, "d")
	router.HandleKey(mock, "i")
	router.HandleKey(mock, "w")
	if mock.DeleteCalls != 1 {
		t.Errorf("Expected 1 Delete call from 'diw', got %d", mock.DeleteCalls)
	}

	// 5. Mode switch to Insert mode ('i')
	router.HandleKey(mock, "i")
	if mock.Mode != 1 {
		t.Errorf("Expected mode 1 (Insert), got %d", mock.Mode)
	}

	// 6. Test typing in Insert mode
	router.HandleKey(mock, "H")
	router.HandleKey(mock, "i")
	if mock.InsertCalls != 2 {
		t.Errorf("Expected 2 InsertCalls, got %d", mock.InsertCalls)
	}

	// 7. Return to Normal mode via <Esc>
	router.HandleKey(mock, "<Esc>")
	if mock.Mode != 0 {
		t.Errorf("Expected mode 0 (Normal), got %d", mock.Mode)
	}

	// 8. Enter Command mode via ':'
	router.HandleKey(mock, ":")
	if mock.Mode != 3 {
		t.Errorf("Expected mode 3 (Command), got %d", mock.Mode)
	}

	// 9. Type command 'w' and press <Enter>
	router.HandleKey(mock, "w")
	if mock.CmdBuffer != "w" {
		t.Errorf("Expected CmdBuffer 'w', got %q", mock.CmdBuffer)
	}

	router.HandleKey(mock, "<Enter>")
	if mock.ExecutedCmd != "w" {
		t.Errorf("Expected ExecutedCmd 'w', got %q", mock.ExecutedCmd)
	}
	if mock.Mode != 0 {
		t.Errorf("Expected mode 0 after command execution, got %d", mock.Mode)
	}
}
