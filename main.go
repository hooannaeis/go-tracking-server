package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Flows struct {
	Flows []Flow `json:"flows"`
}

type Flow struct {
	Id               int              `json:"id"`
	Name             string           `json:"name"`
	Responsibilities []Responsibility `json:"responsibilities"`
	EventKeys        []WhatWhere      `json:"eventKeys"`
}

type Responsibility struct {
	Where string `json:"where"`
	What  string `json:"what"`
	How   string `json:"how"`
}

type WhatWhere struct {
	Where string `json:"where"`
	What  string `json:"what"`
}

func main() {
	// Use an external setup function in order
	// to configure the app in tests as well
	app := Setup()

	// start the application on http://localhost:3000
	log.Fatal(app.Listen(":3000"))
}

func getInitConfig() Flows {
	file, err := ioutil.ReadFile("assets/config-example.json")
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

func Setup() *fiber.App {
	// Initialize a new app
	app := fiber.New()

	app.Get("/*", func(c *fiber.Ctx) error {
		fmt.Println("----------------------")
		flow, claimed := getResponsibleFlow(c)

		if claimed {
			GetEventData(c, flow)
			return c.JSON(flow)
		} else {
			return c.Status(404).SendString("no flow is responsible for this path")
		}
	})

	return app
}

func GetEventData(c *fiber.Ctx, flow Flow) {
	fmt.Println("----------------------")
	for _, eventKey := range flow.EventKeys {
		whereParts := strings.Split(eventKey.Where, ".")
		if whereParts[0] == "Function" {
			switch whereParts[1] {
			case "Hostname":
				fmt.Printf("the %s is %q", eventKey.What, c.Hostname())
			case "Path":
				fmt.Printf("the %s is %q", eventKey.What, c.Path())
			case "IP":
				fmt.Printf("the %s is %q", eventKey.What, c.IP())
			}
		} else if whereParts[0] == "Get" {
			fmt.Printf("the %s is %q", eventKey.What, c.Get(whereParts[1]))
		}
	}
}
