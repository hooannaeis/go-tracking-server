package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Flows are the main object in a configuration file
type Flows struct {
	Flows []Flow `json:"flows"`
}

// Flow describes one specific flow
type Flow struct {
	ID               int                  `json:"id"`
	Name             string               `json:"name"`
	Responsibilities []WhatWhereHowString `json:"responsibilities"`
	EventKeys        []WhatWhereHowString `json:"eventKeys"`
	Actions          []Action             `json:"actions"`
}

// WhatWhereHowString has all optional parameters giving specifics for a Flow-task
type WhatWhereHowString struct {
	Where string `json:"where,omitempty"`
	What  string `json:"what,omitempty"`
	How   string `json:"how,omitempty"`
}

// Action describes the specifics of an action
type Action struct {
	What       string     `json:"what"`
	Where      string     `json:"where,omitempty"`
	HowForward HowForward `json:"howForward,omitempty"`
	Then       []*Action  `json:"then,omitempty"`
}

// HowForward describes what a forward action looks like
type HowForward struct {
	RequestMethod string               `json:"requestMethod"`
	Headers       []WhatWhereHowString `json:"headers,omitempty"`
	Body          []WhatWhereHowString `json:"body,omitempty"`
	Query         []WhatWhereHowString `json:"query,omitempty"`
}

// Event describes all possible parameters that an event can have
type Event struct {
	Hostname string
	Path     string
	RemoteIP string
	Headers  map[string]*ValueString
}

// ValueString only has a value
type ValueString struct {
	value string
}

func main() {
	// Use an external setup function in order
	// to configure the app in tests as well
	app := setup()

	// start the application on http://localhost:3000
	log.Fatal(app.Listen(":3000"))
}

func getInitConfig() Flows {
	file, err := ioutil.ReadFile("assets/config-example-simple.json")
	if err != nil {
		fmt.Println(err)
	}

	flows := Flows{}
	_ = json.Unmarshal([]byte(file), &flows)
	fmt.Println(flows)

	return flows
}

func isThisFlowResponsible(flow Flow, c *fiber.Ctx) bool {
	responsibilitiesToMeet := len(flow.Responsibilities)
	metResponsibilities := 0

	for _, responsibility := range flow.Responsibilities {
		var where string
		parts := strings.Split(responsibility.Where, ".")

		location := parts[0]
		// @todo: add more possibilities
		if location == "request" {
			switch parts[1] {
			case "URL":
				if parts[2] == "Path" {
					where = c.Path()
				} else if parts[2] == "Hostname" {
					where = c.Hostname()
				} else if parts[2] == "RequestURI" {
					where = c.Hostname() + c.Path()
				}
				// @TODO: add option for url-query parameters
			case "Header":
				where = c.Get(parts[2])
			default:
				where = ""
			}
			fmt.Printf("where = %s", where)
		}

		switch responsibility.How {
		case "equalsTrue":
			if where == responsibility.What {
				metResponsibilities++
			}
		case "equalsFalse":
			if where != responsibility.What {
				metResponsibilities++
			}
		case "containsTrue":
			if strings.Contains(where, responsibility.What) {
				metResponsibilities++
			}
		case "containsFalse":
			if !strings.Contains(where, responsibility.What) {
				metResponsibilities++
			}
		}
	}
	return responsibilitiesToMeet == metResponsibilities
}

func getResponsibleFlow(c *fiber.Ctx) (Flow, bool) {
	flows := getInitConfig()
	requestHasBeenClaimed := false
	var claimingFlow Flow

	for _, flow := range flows.Flows {
		isResponsible := isThisFlowResponsible(flow, c)

		// if this flow is responsible AND the request has not
		// been claimed yet, we claim it and stop checking
		if isResponsible && !requestHasBeenClaimed {
			requestHasBeenClaimed = true
			claimingFlow = flow
			break
		}
	}

	if requestHasBeenClaimed {
		fmt.Println("request claimed")
	} else {
		fmt.Println("request not claimed")
	}
	return claimingFlow, requestHasBeenClaimed
}

func setup() *fiber.App {
	// Initialize a new app
	app := fiber.New()

	app.Get("/*", func(c *fiber.Ctx) error {
		fmt.Println("----------------------")
		flow, claimed := getResponsibleFlow(c)

		if claimed {
			event := getEventData(c, flow)
			fmt.Println("----------------------")
			fmt.Println(event)
			fmt.Println("----------------------")

			runActions(flow.Actions, event)

			return c.JSON(flow)
		} else {
			return c.Status(404).SendString("no flow is responsible for this path")
		}
	})

	return app
}

func getEventData(c *fiber.Ctx, flow Flow) Event {
	event := new(Event)

	for _, eventKey := range flow.EventKeys {
		whereParts := strings.Split(eventKey.Where, ".")
		whereMethod := whereParts[0]
		whereKey := whereParts[1]

		// match the keys to their corresponding function-equivalents
		if whereMethod == "Function" {
			switch whereKey {
			case "Hostname":
				event.Hostname = c.Hostname()
			case "Path":
				event.Path = c.Path()
			case "IP":
				event.RemoteIP = c.IP()
			}
		} else if whereMethod == "Get" {
			// check if the current event already has
			// a .Headers mpa
			headers := event.Headers
			if len(headers) == 0 {
				// initialize a map of strings that point to
				// structs of type ValueString
				event.Headers = make(map[string]*ValueString)
			}

			header, headerExists := event.Headers[whereKey]
			if !headerExists {
				header = &ValueString{}
			}
			event.Headers[whereKey] = header
			event.Headers[whereKey].value = c.Get(whereKey)
		}
	}
	return *event
}

func runActions(actions []Action, e Event) {
	for _, action := range actions {
		switch action.What {
		case "forward":
			client := &http.Client{}

			requestURL := makeRequestURL(action, e)
			req, reqErr := http.NewRequest(action.HowForward.RequestMethod, requestURL, nil)
			if reqErr != nil {
				log.Fatal(reqErr)
				return
			}

			for _, headerToAdd := range action.HowForward.Headers {
				req.Header.Add(headerToAdd.Where, headerToAdd.What)
			}
			resp, respErr := client.Do(req)
			if respErr != nil {
				log.Fatal(respErr)
				return
			}

			fmt.Println("forward action", resp)
		case "process":
			fmt.Println("process action")
		}
	}
}

func makeRequestURL(action Action, e Event) string {
	url := action.Where
	for idx, query := range action.HowForward.Query {
		if idx == 0 {
			url += "?"
		} else {
			url += "&"
		}
		url += query.Where + "=" + query.What
	}
	return url
}
