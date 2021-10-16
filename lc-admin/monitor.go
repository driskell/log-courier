/*
 * Copyright 2012-2020 Jason Woods and contributors
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
	"log"
	"time"

	"github.com/driskell/log-courier/lc-admin/views"
	"github.com/driskell/log-courier/lc-lib/admin"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type monitor struct {
	client     *admin.Client
	updateChan chan interface{}
}

func NewMonitor(client *admin.Client) *monitor {
	return &monitor{
		client:     client,
		updateChan: make(chan interface{}),
	}
}

func (m *monitor) Run() error {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	termWidth, termHeight := ui.TerminalDimensions()

	menu := widgets.NewParagraph()
	menu.Border = false
	menu.Text = "[[f]](fg:black,bg:white) Files [[t]](fg:black,bg:white) Transport [[q]](fg:black,bg:white) Quit"
	menu.SetRect(0, 0, termWidth, 1)

	var currentView views.View
	currentView = views.NewProspector(m.client, m.updateChan)
	currentView.SetRect(0, 1, termWidth, termHeight)

	ui.Render(menu, currentView)

	var updateAt <-chan time.Time
	var updatingView views.View

	updatingView = currentView
	go currentView.StartUpdate()

	uiEvents := ui.PollEvents()
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return nil
			case "f":
				currentView = views.NewProspector(m.client, m.updateChan)
				currentView.SetRect(0, 1, termWidth, termHeight)
				if updateAt != nil {
					updatingView = currentView
					go currentView.StartUpdate()
				}
			case "t":
				currentView = views.NewPublisher(m.client, m.updateChan)
				currentView.SetRect(0, 1, termWidth, termHeight)
				if updateAt != nil {
					updatingView = currentView
					go currentView.StartUpdate()
				}
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				menu.SetRect(0, 0, termWidth, 1)
				termWidth = payload.Width
				termHeight = payload.Height
				currentView.SetRect(0, 1, termWidth, termHeight)
				ui.Clear()
				ui.Render(menu, currentView)
			}
		case <-updateAt:
			updateAt = nil
			updatingView = currentView
			go currentView.StartUpdate()
		case resp := <-m.updateChan:
			updatingView.CompleteUpdate(resp)
			if updatingView == currentView {
				updateAt = time.After(time.Second)
			} else {
				updatingView = currentView
				go currentView.StartUpdate()
			}
			ui.Render(menu, currentView)
		}
	}
}
