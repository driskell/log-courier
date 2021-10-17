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
	"fmt"
	"strings"
	"time"

	"github.com/driskell/log-courier/lc-admin/views"
	"github.com/driskell/log-courier/lc-lib/admin"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type monitorViewFactory func(*admin.Client, chan<- interface{}) views.View

type monitorView struct {
	id      string
	factory monitorViewFactory
	key     string
	name    string
	enabled bool
}

type monitor struct {
	client      *admin.Client
	updateChan  chan interface{}
	initialView string
	views       []*monitorView
	viewsByKey  map[string]*monitorView
}

func NewMonitor(client *admin.Client) *monitor {
	initialView := ""
	views := []*monitorView{
		{
			id:      "prospector",
			factory: views.NewProspector,
			key:     "f",
			name:    "Files",
		},
		{
			id:      "receiver",
			factory: views.NewReceiver,
			key:     "r",
			name:    "Receiver",
		},
		{
			id:      "publisher",
			factory: views.NewPublisher,
			key:     "t",
			name:    "Transport",
		},
	}

	viewsByKey := map[string]*monitorView{}
	remoteSummary := client.RemoteSummary()
	for _, view := range views {
		if _, ok := remoteSummary[view.id]; ok {
			view.enabled = true
			if initialView == "" {
				initialView = view.key
			}
		}
		viewsByKey[view.key] = view
	}

	return &monitor{
		client:      client,
		updateChan:  make(chan interface{}),
		initialView: initialView,
		views:       views,
		viewsByKey:  viewsByKey,
	}
}

func (m *monitor) Run() error {
	if m.initialView == "" {
		return fmt.Errorf("remote client does not support monitor mode")
	}
	if err := ui.Init(); err != nil {
		return fmt.Errorf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	termWidth, termHeight := ui.TerminalDimensions()

	menuItems := make([]string, len(m.views))
	for _, view := range m.views {
		if view.enabled {
			menuItems = append(menuItems, fmt.Sprintf("[[%s]](fg:black,bg:white) %s", view.key, view.name))
		}
	}
	menuItems = append(menuItems, "[[q]](fg:black,bg:white) Quit")

	menu := widgets.NewParagraph()
	menu.Border = false
	menu.Text = strings.Join(menuItems, " ")
	menu.SetRect(0, 0, termWidth, 1)

	status := widgets.NewParagraph()
	status.Border = false
	name, version := m.client.RemoteClient()
	status.Text = fmt.Sprintf("%s v%s", name, version)
	status.SetRect(0, termHeight-1, termWidth, termHeight)

	var currentView views.View
	if m.initialView != "" {
		currentView = m.viewsByKey[m.initialView].factory(m.client, m.updateChan)
		currentView.SetRect(0, 1, termWidth, termHeight-1)
	}

	ui.Render(menu, currentView, status)

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
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				menu.SetRect(0, 0, termWidth, 1)
				termWidth = payload.Width
				termHeight = payload.Height
				menu.SetRect(0, 0, termWidth, 1)
				status.SetRect(0, termHeight-1, termWidth, termHeight)
				currentView.SetRect(0, 1, termWidth, termHeight-1)
				ui.Clear()
				ui.Render(menu, currentView, status)
			default:
				if view, ok := m.viewsByKey[e.ID]; ok && view.enabled {
					currentView = view.factory(m.client, m.updateChan)
					currentView.SetRect(0, 1, termWidth, termHeight-1)
					if updateAt != nil {
						updatingView = currentView
						go currentView.StartUpdate()
					}
				}
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
			ui.Render(menu, currentView, status)
		}
	}
}
