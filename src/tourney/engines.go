/*******************************************************************************

 Project: Tourney

 Module: engines
 Description: Engine struct, Protocoler interface, and UCI/WinBoard structs.

 The Engine object has a Protocoler member. Then structs corresponding to UCI
 and Winboard impliment the Protocoler interface. The engine executable itself
 is ran in a goroutine, reader and writer Engine data members read/write the
 executables stdio so other Engine methods can interact with the executable.

TODO:
	-Error checking for it engine path exists and if it opens okay
	-WinBoard protocoler
	-Engines need to take options for hashtable size, multithreading, pondering,
	opening book, and a few other bare minimums.
	-Fix the bug where >> >> >> >> ... keeps looping sometimes.

 Author(s): Andrew Backes, Daniel Sparks
 Created: 7/16/2014

*******************************************************************************/

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// helper:
type rec struct {
	timestamp time.Time
	data      string
}

/*******************************************************************************

	General Engine Functionality:

*******************************************************************************/

type Engine struct {
	//Public:
	Name string
	Path string // file location
	// TODO: case sensitive issues with protocol?
	Protocol string            // = "UCI" or "WINBOARD" (TODO: Auto)
	Options  map[string]string // initialized in E.Initialize()
	MD5      string

	//Private:
	reader *bufio.Reader
	writer *bufio.Writer
	logbuf *string

	protocol         Protocoler         // should = UCI{} or WINBOARD{}
	supportedOptions map[string]Setting // decided after the engine loads and says what it supports.
}

// this struct may need to change depending on how winboard works:
type Setting struct {
	Value string
	Type  string
	Min   string
	Max   string
}

func (E *Engine) ValidateEngineFile() error {
	// First decides if the file exists.
	// Compares the checksum to the md5 sum that is stored in memory.
	// If nothing has been stored, then it saves this checksum.
	// Returns true when they match or it was previously blank.
	// Primarly used when transfering a file to a worker.

	// Existence:
	if _, err := os.Stat(E.Path); os.IsNotExist(err) {
		return err
	}

	// Check sum:
	if checksum, err := GetMD5(E.Path); err != nil {
		return err
	} else {
		if E.MD5 == "" {
			// md5 has not been previously checked
			E.MD5 = checksum
		} else if E.MD5 != checksum {
			return errors.New("MD5 mismatch")
		}
	}

	return nil
}

func (E *Engine) Log(label string, record rec) {
	*E.logbuf += fmt.Sprintln("[" + record.timestamp.Format("01/02/2006 15:04:05.000") + "][" + E.Name + "][" + label + "]" + record.data)

	//DEBUG ONLY:
	//fmt.Println("[" + record.timestamp.Format("01/02/2006 15:04:05.000") + "][" + E.Name + "][" + label + "]" + record.data)
}

// Send a command to the engine:
func (E *Engine) Send(s string) error {
	E.Log("<-", rec{time.Now(), s})
	E.writer.WriteString(fmt.Sprintln(s)) // hopefully the line return is OS specific here.
	E.writer.Flush()
	//fmt.Print("->", fmt.Sprintln(s))
	return nil
}

// Set the engine up to be ready to think on its first move:
func (E *Engine) Start(logbuffer *string) error {
	E.logbuf = logbuffer

	// Decide which protocol to use:
	// TODO: add some autodetect code here
	if strings.ToUpper(E.Protocol) == "UCI" {
		E.protocol = &UCI{}
	} else if strings.ToUpper(E.Protocol) == "WINBOARD" {
		E.protocol = &WINBOARD{}
	}

	cmd := exec.Command(E.Path)

	// Setup the pipes to communicate with the engine:
	StdinPipe, errIn := cmd.StdinPipe()
	if errIn != nil {
		return errors.New("Error Initializing Engine: can not establish in pipe.")
	}
	StdoutPipe, errOut := cmd.StdoutPipe()
	if errOut != nil {
		return errors.New("Error Initializing Engine: can not establish out pipe.")
	}
	E.writer, E.reader = bufio.NewWriter(StdinPipe), bufio.NewReader(StdoutPipe)

	// Start the engine:
	started := make(chan struct{})
	errChan := make(chan error)
	go func() {
		// Question: Does this force the engine to run in its own thread?
		if err := cmd.Start(); err != nil {
			errChan <- err
			return
			//return errors.New("Error executing " + E.Path + " - " + err.Error())
		}
		close(started)
	}()
	select {
	case <-started:
	case e := <-errChan:
		return errors.New("Error executing " + E.Path + " - " + e.Error())
	}

	// Get the engine ready:
	if err := E.Initialize(); err != nil {
		E.Log("ERROR", rec{time.Now(), "Initializing Engine: " + err.Error()})
		return err
	}

	/*
		// DEBUG:
		for k, v := range E.options {
			E.Log("OPTION", rec{time.Now(), fmt.Sprint("Registering engine option ", k, v)})
		}
	*/

	E.NewGame()

	// Setup up for when the engine exits:
	go func() {
		cmd.Wait()
		//TODO: add some confirmation that the engine has terminated correctly.
	}()

	return nil
}

// Send the first commands to the engine and recieves what options/features the engine supports
func (E *Engine) Initialize() error {

	s, r := E.protocol.Initialize()
	E.Send(s)
	var output string
	var err error

	output, _, err = E.Recieve(r, 2000)
	if err != nil {
		return err
	}

	// Listen to what options the engine says it supports.
	E.supportedOptions = make(map[string]Setting)
	E.protocol.RegisterEngineOptions(output, E.supportedOptions)

	return nil
}

// Recieve and process commands until a certain command is recieved
// or after the timeout (milliseconds) is achieved.
// Returns: engine output, lapsed Time, error
func (E *Engine) Recieve(untilCmd string, timeout int64) (string, time.Duration, error) {

	//var err error
	var output string //engine's output

	// Set up the Timer:
	startTime := time.Now()

	// Start recieving from the engine:
	for {
		recieved := make(chan rec, 1)
		errChan := make(chan error, 1)

		//TODO: Redesign neccessary
		go func() {
			// TODO: need a better idea here. ReadString() could hault this goroutine.
			if nextline, err := E.reader.ReadString('\n'); err == nil {
				recieved <- rec{time.Now(), nextline}
			} else {
				errChan <- err
			}
		}()

		// Since the Timer and the reader are in goroutines, wait for:
		// (1) Something from the engine, (2) Too much Time to pass. (3) An error
		select {
		case line := <-recieved:
			// keep track of the total output from the engine:
			output += line.data

			// Take off line return bytes:
			line.data = strings.Trim(line.data, "\r\n") // for windows
			line.data = strings.Trim(line.data, "\n")   // for *nix/bsd

			// Log this line of engine output:
			E.Log("->", line)

			// Check if the recieved command is the one we are waiting for:
			for _, v := range strings.Split(line.data, " ") {
				if v == untilCmd {
					return output, line.timestamp.Sub(startTime), nil
				}
			}

		case <-time.After(time.Duration(timeout) * time.Millisecond):
			description := "Timed out waiting for engine to respond."
			E.Log("ERROR", rec{time.Now(), description})
			return output, time.Now().Sub(startTime), errors.New(description)

		case e := <-errChan:
			description := "Error recieving from engine: " + e.Error()
			E.Log("ERROR", rec{time.Now(), description})
			return output, time.Now().Sub(startTime), errors.New(description)

		}
	}
	return output, time.Now().Sub(startTime), nil //this should never occur
}

func (E *Engine) NewGame() error {
	//E.protocol.New(E.reader, E.writer)
	E.Send(E.protocol.NewGame())
	return nil
}

// The engine should close itself:
func (E *Engine) Shutdown() error {
	// TODO: add confirmation that the engine has shut down correctly
	E.Send(E.protocol.Quit())
	return nil
}

// The engine should decide what move it wants to make:
func (E *Engine) Move(timers [2]int64, MovesToGo int64, EngineColor Color) (Move, time.Duration, error) {
	s, r := E.protocol.Move(timers, MovesToGo, EngineColor)
	E.Send(s)
	max := timers[WHITE]
	if timers[BLACK] > max {
		max = timers[BLACK]
	}
	response, t, err := E.Recieve(r, max+1000)

	if err != nil {
		return Move{}, t, err
	}

	// figure out what move was picked:
	return E.protocol.ExtractMove(response), t, nil
}

// The engine should set its internal Board to adjust for the Moves far in the game
func (E *Engine) Set(movesSoFar []Move) error {
	s := E.protocol.SetBoard(movesSoFar)
	err := E.Send(s)
	return err
}

func (E *Engine) Ping() error {
	s, r := E.protocol.Ping(1)
	E.Send(s)
	_, _, err := E.Recieve(r, -1)
	return err
}

/*******************************************************************************

	Protocol Specific:

*******************************************************************************/

type Protocoler interface {
	Initialize() (string, string)
	Quit() string
	Move(timers [2]int64, MovesToGo int64, EngineColor Color) (string, string)
	SetBoard(moveSoFar []Move) string
	NewGame() string
	Ping(int) (string, string)

	ExtractMove(string) Move
	RegisterEngineOptions(string, map[string]Setting)
}

/*******************************************************************************

	UCI:

*******************************************************************************/

type UCI struct{}

func (U UCI) Ping(N int) (string, string) {
	return "isready", "readyok"
}

func (U UCI) Initialize() (string, string) {
	// (command to send),(command to recieve)
	return "uci", "uciok"
}

func (U UCI) NewGame() string {
	return "ucinewgame"
}

func (U UCI) Quit() string {
	return "quit"
}

func (U UCI) Move(Timer [2]int64, MovesToGo int64, EngineColor Color) (string, string) {
	goString := "go"

	if Timer[WHITE] > 0 {
		goString += " wtime " + strconv.FormatInt(Timer[WHITE], 10)
	}
	if Timer[BLACK] > 0 {
		goString += " btime " + strconv.FormatInt(Timer[BLACK], 10)
	}
	if MovesToGo > 0 {
		goString += " movestogo " + strconv.FormatInt(MovesToGo, 10)
	}

	return goString, "bestmove"
}

func (U UCI) SetBoard(movesSoFar []Move) string {
	var ml []string

	for _, m := range movesSoFar {
		ml = append(ml, m.Algebraic)
	}

	var pos string
	if len(movesSoFar) > 0 {
		pos = "position startpos moves " + strings.Join(ml, " ")
	} else {
		pos = "position startpos"
	}

	return pos
}

func (U UCI) RegisterEngineOptions(output string, options map[string]Setting) {
	// TODO: some engines have two word setting keys
	// 		 ex: senpai: option name Log File type check default false
	// TODO: case sensitive

	if output == "" {
		return
	}

	output = strings.Replace(output, "\r", "", -1) // remove the \r after the \n\r
	lines := strings.Split(output, "\n")
	for i, _ := range lines {
		newSettingLabel := ""
		newSetting := Setting{}
		words := strings.Split(lines[i], " ")
		// double check that this line has option information on it:
		if strings.ToLower(words[0]) != "option" {
			continue
		}
		// Process the option information:
		for j := 0; j < len(words)-1; j++ {
			switch strings.ToLower(words[j]) {
			case "name":
				newSettingLabel = words[j+1]
			case "type":
				newSetting.Type = words[j+1]
			case "default":
				newSetting.Value = words[j+1]
			case "min":
				newSetting.Min = words[j+1]
			case "max":
				newSetting.Max = words[j+1]
			}
		}
		options[newSettingLabel] = newSetting
	}
}

func (U UCI) ExtractMove(output string) Move {

	// TODO: REFACTOR: this replace also happens in Engine.Recieve()
	output = strings.Replace(output, "\n\r", " ", -1)
	output = strings.Replace(output, "\n", " ", -1)
	words := strings.Split(output, " ")

	// Helper function:
	LastValueOf := func(key string) string {
		//returns the word after the word given as an arg
		for i := len(words) - 1; i >= 0; i-- {
			if words[i] == key {
				if i+1 <= len(words)-1 {
					return words[i+1]
				}
			}
		}
		return ""
	}

	d, _ := strconv.Atoi(LastValueOf("depth")) // TODO: error handlingL what if this isnt an int!?
	t, _ := strconv.Atoi(LastValueOf("time"))  // TODO: doing this way may only give the time for this depth
	skey := LastValueOf("score")               // ex: score cp 112   but it could be:   score mate 7

	var sval int
	if skey == "cp" {
		sval, _ = strconv.Atoi(LastValueOf(skey))
	} else if skey == "mate" {
		sval, _ = strconv.Atoi(LastValueOf(skey))
		sval = MateIn(sval)
	}

	return (Move{
		Algebraic: LastValueOf("bestmove"),
		Depth:     d,
		Time:      t,
		Score:     sval,
	})

}

/*******************************************************************************

	WINBOARD:

*******************************************************************************/

type WINBOARD struct {
	features map[string]string
}

func (W WINBOARD) Initialize() (string, string) {
	return "xboard\nprotover 2", "done=1"
}

func (W WINBOARD) NewGame() string {
	return "new\nrandom\npost\nhard\neasy\ncomputer"
}

func (W *WINBOARD) SetBoard(movesSoFar []Move) string {

	// DEBUG:
	//fmt.Print("\nW.features[usermove]='", W.features["usermove"], "'\n\n")

	var pos string

	// Determine if this is the first move this engine will be thinking on:
	movesOutOfBook := 0
	pos = "force\n"
	for i, _ := range movesSoFar {
		if v := W.features["usermove"]; v == "1" {
			pos += "usermove "
		}
		pos += movesSoFar[i].Algebraic + "\n"
		if movesSoFar[i].Comment != BOOKMOVE {
			movesOutOfBook++
		}
	}

	// when there is more than one move out of the book, dont play the opening:
	if movesOutOfBook > 1 {
		pos = "force\n"
		if v := W.features["usermove"]; v == "1" {
			pos += "usermove "
		}
		pos += movesSoFar[len(movesSoFar)-1].Algebraic //only the last move is needed
	}

	return pos
}

func (W WINBOARD) Ping(N int) (string, string) {
	return "ping" + strconv.Itoa(N), "pong" + strconv.Itoa(N)
}

func (W WINBOARD) Quit() string {
	return "quit"
}

func (W *WINBOARD) Move(Timer [2]int64, MovesToGo int64, EngineColor Color) (string, string) {
	var goString string

	goString += "time " + strconv.FormatInt(Timer[EngineColor], 10) + "\n"
	goString += "otim " + strconv.FormatInt(Timer[[]int{1, 0}[EngineColor]], 10) + "\n"
	if W.features["colors"] == "1" {
		goString += []string{"white\n", "black\n"}[EngineColor]
	}
	goString += "go"

	return goString, "move"
}

func (W WINBOARD) ExtractMove(output string) Move {

	// TODO: REFACTOR: this replace also happens in Engine.Recieve()
	output = strings.Replace(output, "\r", " ", -1)
	lines := strings.Split(output, "\n")

	var d, s, t int
	var m string
	for _, line := range lines {
		fields := strings.Fields(line)
		if fields[0] == "move" {
			if len(fields) > 1 {
				m = fields[1]
			}
			break
		}
		// ply score time nodes pv
		// ex: 6    11       0      5118 Qd5 9.Bf4 Nc6 10.e3 Bg4 11.a3 [TT]
		// ex: 8&     66    1    20536   d1e2  e8e7  e2e3  e7e6  e3d4  g7g5  a2a4  f7f5
		if len(fields) >= 4 {
			if isNumber(fields[0]) {
				d, _ = strconv.Atoi(fields[0])
			} else if isNumber(fields[0][:len(fields[0])-1]) {
				// accounts for the case of 8& or 8.
				d, _ = strconv.Atoi(fields[0][:len(fields[0])-1])
			}
			if isNumber(fields[1]) {
				s, _ = strconv.Atoi(fields[1])
			}
			if isNumber(fields[2]) {
				t, _ = strconv.Atoi(fields[2])
			}
		}
	}

	return (Move{
		Algebraic: m,
		Depth:     d,
		Time:      t,
		Score:     s,
	})
}

func (W *WINBOARD) RegisterEngineOptions(output string, options map[string]Setting) {

	// helper. Splits based on spaces not inside quotes:
	nonQuotedWordSplit := func(ln string) []string {
		r := []string{}
		quoted := false
		var b int
		for i, v := range ln {
			if string(v) == "\"" {
				quoted = !quoted
			}
			if string(v) == " " && !quoted || i == len(ln)-1 {
				r = append(r, strings.Trim(ln[b:i+1], " "))
				b = i + 1
			}
		}
		return r
	}
	// ***

	W.features = make(map[string]string) // init for local struct use
	W.setFeaturesToDefault()

	output = strings.Replace(output, "\r", "", -1)
	lines := strings.Split(output, "\n")

	for _, v := range lines {
		if strings.HasPrefix(v, "feature") {
			v = v[len("feature "):]
		} else {
			continue
		}

		pairs := nonQuotedWordSplit(v)
		for j, _ := range pairs {
			p := strings.Split(pairs[j], "=")
			if p[0] != "option" { // TEMPORARY
				if len(p) > 1 {
					W.features[p[0]] = p[1]
					//fmt.Println("accepted", p[0], p[1])
				}
			}
		}
	}
}

// Sets the feature list to the Winboard defaults:
func (W *WINBOARD) setFeaturesToDefault() {

	// Winboard/xboard default values:
	W.features["ping"] = "0"      //ping (boolean, default 0, recommended 1)
	W.features["setboard"] = "0"  //setboard (boolean, default 0, recommended 1)
	W.features["playother"] = "0" //playother (boolean, default 0, recommended 1)
	W.features["san"] = "0"       //san (boolean, default 0)
	W.features["usermove"] = "0"  //usermove (boolean, default 0)
	W.features["time"] = "1"      //time (boolean, default 1, recommended 1)
	W.features["draw"] = "1"      //draw (boolean, default 1, recommended 1)
	W.features["sigint"] = "1"    //sigint (boolean, default 1)
	W.features["sigterm"] = "1"   //sigterm (boolean, default 1)
	W.features["reuse"] = "1"     //reuse (boolean, default 1, recommended 1)
	W.features["analyze"] = "1"   //analyze (boolean, default 1, recommended 1)
	W.features["colors"] = "1"    //colors (boolean, default 1, recommended 0)
	W.features["ics"] = "0"       //ics (boolean, default 0)
	W.features["pause"] = "0"     // pause (boolean, default 0)
	W.features["debug"] = "0"     //debug (boolean, default 0)
	W.features["memory"] = "0"    //memory (boolean, default 0)
	W.features["smp"] = "0"       //smp (boolean, default 0)
	W.features["exclude"] = "0"   //exclude (boolean, default 0)
	W.features["setscore"] = "0"  //setscore (boolean, default 0)
	W.features["highlight"] = "0" //highlight (boolean, default 0)

}
