/*******************************************************************************

 Project: 		Tourney
 Module: 		consoleui
 Created: 		9/27/2015
 Author(s): 	Andrew Backes
 Description: 	Viewer for the console.

*******************************************************************************/

package main

import (
	"bufio"
	"fmt"
	"os"
)

func ConsoleUI(controller *Controller) {

	inputReader := bufio.NewReader(os.Stdin)

	var prompt string
	for !controller.Stopped() {
		prompt = ""
		fmt.Print(prompt)
		line, _ := inputReader.ReadString('\n')
		controller.Enque(line)
	}
}