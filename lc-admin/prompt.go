/*
 * Copyright 2014-2015 Jason Woods.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"
)

type prompt struct {
	commandProcessor commandProcessor
}

func (p *prompt) run() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	fmt.Printf("> ")

	commandChan := make(chan string)
	go func() {
		var discard bool
		reader := bufio.NewReader(os.Stdin)
		for {
			line, prefix, err := reader.ReadLine()
			if err != nil {
				if err == io.EOF {
					fmt.Printf("exit\n")
				} else {
					fmt.Printf("\nError: %s\n", err)
				}
				commandChan <- "exit"
			} else if prefix {
				discard = true
			} else if discard {
				fmt.Printf("\nLine too long!\n> ")
				discard = false
			} else {
				commandChan <- string(line)
			}
		}
	}()

CommandLoop:
	for {
		select {
		case command := <-commandChan:
			switch command {
			case "":
			case "exit":
				break CommandLoop
			default:
				p.commandProcessor.ProcessCommand(command)
			}
			fmt.Printf("> ")
		case <-signalChan:
			break CommandLoop
		}
	}
}

func (p *prompt) argsCommand(args []string, watch bool) bool {
	var signalChan chan os.Signal

	if watch {
		signalChan = make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
	}

WatchLoop:
	for {
		if !p.commandProcessor.ProcessCommand(strings.Join(args, " ")) {
			if !watch {
				return false
			}
		}

		if !watch {
			break
		}

		// Gap between repeats
		fmt.Printf("\n")

		select {
		case <-signalChan:
			break WatchLoop
		case <-time.After(time.Second):
		}
	}

	return true
}
