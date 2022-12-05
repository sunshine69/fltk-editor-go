package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cjoudrey/gluahttp"
	"github.com/kohkimakimoto/gluayaml"
	"github.com/pwiecz/go-fltk"
	"github.com/sunshine69/gluare"
	u "github.com/sunshine69/golang-tools/utils"
	gopherjson "github.com/sunshine69/gopher-json"
	lua "github.com/yuin/gopher-lua"
)

const (
	WIDGET_HEIGHT  = 400
	WIDGET_PADDING = 0
	WIDGET_WIDTH   = 800
)

type EditorApp struct {
	Win              *fltk.Window
	TextBuffer       *fltk.TextBuffer
	TextEditor       *fltk.TextEditor
	FileName         string
	IsChanged        bool
	WrapMode         fltk.WrapMode
	ProcessingDialog *textProcessingDialog
}

func (app *EditorApp) BuildGUI() {
	fltk.InitStyles()
	fltk.SetScheme("gtk+")

	app.Win = fltk.NewWindow(WIDGET_WIDTH, WIDGET_HEIGHT)

	app.Win.SetLabel("TextEditor")
	app.Win.Resizable(app.Win)

	hpack := fltk.NewPack(WIDGET_PADDING, WIDGET_PADDING, app.Win.W(), WIDGET_HEIGHT)
	hpack.SetType(fltk.VERTICAL)
	hpack.SetSpacing(WIDGET_PADDING)

	menuBar := fltk.NewMenuBar(0, 0, app.Win.W(), 20)
	menuBar.SetType(uint8(fltk.FLAT_BOX))
	menuBar.Activate()
	menuBar.AddEx("File", fltk.ALT+'f', nil, fltk.SUBMENU)
	menuBar.AddEx("File/&New", fltk.CTRL+'n', app.callbackMenuFileNew, fltk.MENU_VALUE)
	menuBar.AddEx("File/&Open", fltk.CTRL+'o', app.callbackMenuFileOpen, fltk.MENU_VALUE)
	menuBar.AddEx("File/O&pen As New", fltk.CTRL+'O', app.callbackMenuFileOpenAsNew, fltk.MENU_VALUE)
	menuBar.AddEx("File/&Save", fltk.CTRL+'s', app.callbackMenuFileSave, fltk.MENU_VALUE)
	menuBar.Add("File/Save &As", app.callbackMenuFileSaveAs)
	menuBar.AddEx("File/Save+Close", fltk.CTRL+'X', app.callbackMenuFileSaveClose, fltk.MENU_VALUE)
	menuBar.AddEx("File/Insert", fltk.CTRL+'i', app.callbackMenuFileInsert, fltk.MENU_VALUE)
	menuBar.AddEx("File/Exit", fltk.CTRL+'q', app.callbackMenuFileExit, fltk.MENU_VALUE)

	menuBar.AddEx("Edit", fltk.ALT+'e', nil, fltk.SUBMENU)
	menuBar.AddEx("Edit/&Toggle Wrap mode", fltk.ALT+'z', app.callbackMenuEditToggleWrapMode, fltk.MENU_VALUE)
	menuBar.AddEx("Edit/&Find Replace", fltk.CTRL+'f', app.callbackMenuEditFind, fltk.MENU_VALUE)
	// menuBar.AddEx("Edit/&Paste", fltk.CTRL+'c', app.callbackMenuEditCopy, fltk.MENU_VALUE)
	menuBar.AddEx("Help", 0, nil, fltk.SUBMENU)
	menuBar.Add("Help/&About", app.callbackMenuHelpAbout)
	menuBar.Add("Help/&Test", app.callbackMenuHelpTest)

	app.TextBuffer = fltk.NewTextBuffer()
	app.TextEditor = fltk.NewTextEditor(0, 0, app.Win.W(), app.Win.H()-20)
	app.TextEditor.SetBuffer(app.TextBuffer)

	app.TextEditor.SetCallbackCondition(fltk.WhenChanged)
	app.TextEditor.SetCallback(func() {
		app.IsChanged = true
	})
	app.TextEditor.Parent().Resizable(app.TextEditor)

	app.Win.End()
	app.IsChanged = false
}

func (app *EditorApp) callbackMenuHelpTest() {
	app.TextEditor.Copy()
	app.TextEditor.Paste()
}

func NewEditor() EditorApp {
	myapp := EditorApp{}
	myapp.BuildGUI()
	myapp.Win.Show()
	return myapp
}

func RunLuaFile(luaFileName string) string {
	old := os.Stdout // keep backup of the real stdout

	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	L := lua.NewState()
	defer L.Close()
	L.PreloadModule("re", gluare.Loader)
	L.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	L.PreloadModule("yaml", gluayaml.Loader)
	L.PreloadModule("json", gopherjson.Loader)

	err := L.DoFile(luaFileName)
	if err := u.CheckErrNonFatal(err, "Lua DoFile"); err != nil {
		fmt.Print(err.Error())
	}

	w.Close()
	os.Stdout = old
	out := <-outC
	return out
}

func (app *EditorApp) callbackMenuFileNew() {
	NewEditor()
}

type textProcessingDialog struct {
	win                           *fltk.Window
	app                           *EditorApp
	input, replaceText            *fltk.Input
	icase, cmd, new               *fltk.CheckButton
	bt_find, bt_repl, bt_repl_all *fltk.Button
	startPos, endPos              int
	isBackward                    *fltk.CheckButton
	scriptPath                    string
}

func NewTextProcessingDialog(app *EditorApp) textProcessingDialog {
	w := fltk.NewWindow(460, 105)
	w.Resizable(w)
	appTitles := strings.Split(app.Win.Label(), string(os.PathSeparator))
	filename := u.Ternary(len(appTitles) == 1, appTitles[0], appTitles[len(appTitles)-1])
	w.SetLabel("Search/Replace - " + filename.(string))
	input := fltk.NewInput(10, 10, 220, 25, "")
	input.SetTooltip("input text or command in command mode")
	icase := fltk.NewCheckButton(240, 10, 25, 25, "icase")
	icase.SetTooltip("togle case sensitive in search mode")
	cmd := fltk.NewCheckButton(310, 10, 25, 25, "cmd")
	cmd.SetTooltip("command mode. If enable, then input will take commands to process. Supported commands:\ngopher-lua - Run the text in editor as a lua script\nAny External system command will take the text/selection and process it if the replace out put has string <CMD_OUTPUT>\nType a golang regex and the replacemant text is any wil do search / replace usin golang regex")
	new := fltk.NewCheckButton(370, 10, 25, 25, "new")
	new.SetTooltip("output to new editor instead of replace the current selection")

	replace := fltk.NewInput(10, 40, 220, 25, "")
	replace.SetTooltip("replace text")
	isBackward := fltk.NewCheckButton(240, 40, 25, 25, "backward")

	bt_find := fltk.NewButton(10, 70, 120, 25, "Find")
	bt_find.SetTooltip("")
	bt_repl := fltk.NewButton(140, 70, 120, 25, "Replace")
	bt_repl.SetTooltip("")
	bt_repl_all := fltk.NewButton(280, 70, 120, 25, "Repl all")
	bt_repl_all.SetTooltip("")

	w.End()
	d := textProcessingDialog{}
	d.app, d.win, d.input, d.replaceText, d.icase, d.cmd, d.new, d.isBackward, d.bt_find, d.bt_repl, d.bt_repl_all = app, w, input, replace, icase, cmd, new, isBackward, bt_find, bt_repl, bt_repl_all

	cmd.SetCallback(func() {
		if cmd.Value() {
			if d.replaceText.Value() == "" {
				d.replaceText.SetValue("<CMD_OUTPUT>")
			}
			bt_find.SetLabel("Exec")
			bt_repl.SetLabel("Load script")
			bt_repl_all.SetLabel("Clear script")
		} else {
			bt_find.SetLabel("Find")
			bt_repl.SetLabel("Replace")
			bt_repl_all.SetLabel("Repl All")
		}
	})
	bt_find.SetCallback(d.FindExec)
	bt_repl.SetCallback(d.ReplaceLoad)

	bt_repl_all.SetCallback(d.ReplaceAll)
	return d
}

func (d *textProcessingDialog) FindExec() {
	if d.cmd.Value() {
		d.Exec()
	} else {
		d.Find()
	}
}

func (d *textProcessingDialog) Exec() {
	app := d.app
	var text string = app.TextBuffer.GetSelectionText()
	var outStr string
	isSelection := true
	if text == "" {
		text = app.TextBuffer.Text()
		isSelection = false
		fmt.Println("not any selection")
	}
	if d.replaceText.Value() == "<CMD_OUTPUT>" {
		d.ExecCodeSnippet()
	} else {
		keyword := strings.TrimSpace(d.input.Value())
		if ptn, e := regexp.Compile(keyword); e == nil {
			outStr = ptn.ReplaceAllString(text, d.replaceText.Value())
		}
		if !d.new.Value() {
			if isSelection {
				d.app.TextBuffer.ReplaceSelection(outStr)
			} else {
				d.app.TextBuffer.SetText(outStr)
			}
		} else {
			newEditor := NewEditor()
			newEditor.TextBuffer.SetText(outStr)
			newEditor.Win.Show()
		}
	}
}

func (d *textProcessingDialog) ExecCodeSnippet() {
	keyword := d.input.Value()
	_tmpF, _ := ioutil.TempFile("", fmt.Sprintf("fltk-texteditor-*%s", filepath.Ext(d.app.FileName)))
	text, isSelection := d.app.TextBuffer.GetSelectionText(), true
	if text == "" {
		text = d.app.TextBuffer.Text()
		isSelection = false
	}
	_tmpF.Write([]byte(text))
	err := _tmpF.Close()
	u.CheckErrNonFatal(err, "run-command can not close tmp file")
	cmdText := fmt.Sprintf("%s %s %s", keyword, d.scriptPath, _tmpF.Name())

	commandList := strings.Fields(cmdText)
	var outStr string
	if commandList[0] == "gopher-lua" {
		// Use internal lua VM to run the code
		if d.scriptPath == "" {
			outStr = RunLuaFile(_tmpF.Name())
		} else {
			// Inside the script file, get the data file from the env and process it
			os.Setenv("DATA_FILE", _tmpF.Name())
			outStr = RunLuaFile(d.scriptPath)
		}
	} else {
		cmd := exec.Command(commandList[0], commandList[1:]...)
		cmd.Env = append(os.Environ())
		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("DEBUG E %v\n", err)
		}
		outStr = string(stdoutStderr)
	}
	os.Remove(_tmpF.Name())

	if !d.new.Value() {
		if isSelection {
			d.app.TextBuffer.ReplaceSelection(outStr)
		} else {
			d.app.TextBuffer.SetText(outStr)
		}
	} else {
		newEditor := NewEditor()
		newEditor.TextBuffer.SetText(outStr)
		newEditor.Win.Show()
	}
}

func (d *textProcessingDialog) Find() {
	txt := d.input.Value()
	app := d.app
	d.startPos = d.app.TextEditor.TextDisplay.GetInsertPosition()
	pos := app.TextBuffer.Search(d.startPos, txt, d.isBackward.Value(), !d.icase.Value())
	if pos != -1 {
		d.endPos = pos + len(txt)
		app.TextBuffer.Select(pos, d.endPos)
		if d.isBackward.Value() {
			d.startPos = pos - len(txt)
			app.TextEditor.SetInsertPosition(d.startPos)
		} else {
			d.startPos = d.endPos
			app.TextEditor.SetInsertPosition(d.endPos)
		}
		app.TextEditor.ShowInsertPosition()
	} else {
		fltk.MessageBox("INFO", "Reaching end of buffer, will reset start position")
		if d.isBackward.Value() {
			for app.TextEditor.TextDisplay.MoveDown() {

			}
			d.startPos = app.TextEditor.TextDisplay.GetInsertPosition()
		} else {
			app.TextEditor.SetInsertPosition(0)
		}
	}
}

func (d *textProcessingDialog) ReplaceLoad() {
	if d.bt_repl.Label() == "Replace" {
		d.app.TextBuffer.ReplaceSelection(d.replaceText.Value())
	} else {
		fchooser := fltk.NewFileChooser("", "*.*", fltk.FileChooser_SINGLE, "Select script file")
		fchooser.Popup()
		fnames := fchooser.Selection()
		if len(fnames) == 1 {
			d.scriptPath = fnames[0]
		}
	}
}

func (d *textProcessingDialog) ReplaceAll() {
	if d.cmd.Value() {
		d.scriptPath = ""
		return
	}
	txt := strings.TrimSpace(d.input.Value())
	app := d.app
	replText := d.replaceText.Value()
	d.startPos = d.app.TextEditor.TextDisplay.GetInsertPosition()
	for {
		pos := app.TextBuffer.Search(d.startPos, txt, false, !d.icase.Value())
		if pos == -1 {
			break
		}
		d.endPos = pos + len(txt)
		app.TextBuffer.Select(pos, d.endPos)
		d.app.TextBuffer.ReplaceSelection(replText)
		d.startPos = pos + len(replText)
	}
}

func (app *EditorApp) callbackMenuEditFind() {
	if app.ProcessingDialog == nil {
		ProcessingDialog := NewTextProcessingDialog(app)
		app.ProcessingDialog = &ProcessingDialog
	}
	app.ProcessingDialog.win.Show()
}

func (app *EditorApp) callbackMenuEditToggleWrapMode() {
	if app.WrapMode == fltk.WRAP_NONE {
		app.WrapMode = fltk.WRAP_AT_BOUNDS
	} else {
		app.WrapMode = fltk.WRAP_NONE
	}
	app.TextEditor.TextDisplay.SetWrapMode(app.WrapMode)
}

func (app *EditorApp) callbackMenuFileOpenAsNew() {
	fChooser := fltk.NewFileChooser("./", "*.*", fltk.FileChooser_SINGLE, "Open text file")
	defer fChooser.Destroy()
	fChooser.Popup()
	fnames := fChooser.Selection()
	myapp := NewEditor()
	if len(fnames) == 1 {
		myapp.LoadFile(fnames[0])
	}
}

func (app *EditorApp) LoadFile(filename string) {
	textByte, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	app.TextBuffer.SetText(string(textByte))
	app.FileName = filename
	app.Win.SetLabel(filename)
}

func (app *EditorApp) callbackMenuFileOpen() {
	fChooser := fltk.NewFileChooser("./", "*.*", fltk.FileChooser_SINGLE, "Open text file")
	defer fChooser.Destroy()
	fChooser.Popup()
	fnames := fChooser.Selection()
	if len(fnames) == 1 {
		app.LoadFile(fnames[0])
	}
}

func (app *EditorApp) callbackMenuFileInsert() {
	fChooser := fltk.NewFileChooser("./", "*.*", fltk.FileChooser_SINGLE, "Open text file")
	defer fChooser.Destroy()
	fChooser.Popup()
	fnames := fChooser.Selection()
	if len(fnames) == 1 {
		textByte, err := os.ReadFile(fnames[0])
		if u.CheckErrNonFatal(err, "ReadFile") != nil {
			fltk.MessageBox("ERROR reading file", err.Error())
			return
		}
		app.TextEditor.InsertText(string(textByte))
	}
}

func (app *EditorApp) callbackMenuFileSaveClose() {
	app.callbackMenuFileSave()
	app.Win.Destroy()
}

func (app *EditorApp) callbackMenuFileExit() {
	app.Win.Destroy()
	os.Exit(0)
}

func (app *EditorApp) callbackMenuFileSave() {
	if app.IsChanged {
		info, _ := os.Stat(app.FileName)
		os.WriteFile(app.FileName, []byte(app.TextBuffer.Text()), info.Mode())
		app.IsChanged = false
	}
}

func (app *EditorApp) callbackMenuFileSaveAs() {
	fChooser := fltk.NewFileChooser("./", "*.*", fltk.FileChooser_CREATE, "Enter/Select file name")
	defer fChooser.Destroy()
	fChooser.Popup()
	fnames := fChooser.Selection()
	if len(fnames) == 1 {
		os.WriteFile(fnames[0], []byte(app.TextBuffer.Text()), 0640)
		app.IsChanged = false
		app.FileName = fnames[0]
	}
}

func (app *EditorApp) callbackMenuHelpAbout() {
	fltk.MessageBox("About", "Text Editor nd Processor")
}

func main() {
	mingw64RootDir := flag.String("create-win-bundle", "", "Pass the mingw64 root dir to create the windows-bundle package")
	flag.Parse()

	if *mingw64RootDir != "" {
		CreateWinBundle(*mingw64RootDir)
		return
	}
	NewEditor()
	fltk.Run()
}
