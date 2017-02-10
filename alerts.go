package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	mkr "github.com/mackerelio/mackerel-client-go"
	"github.com/mackerelio/mkr/logger"
	"gopkg.in/urfave/cli.v2"
)

var commandAlerts = &cli.Command{
	Name:  "alerts",
	Usage: "Retrieve/Close alerts",
	Description: `
    Retrieve/Close alerts. With no subcommand specified, this will show all alerts.
    Requests APIs under "/api/v0/alerts". See https://mackerel.io/api-docs/entry/alerts .
`,
	Action: doAlertsRetrieve,
	Subcommands: []*cli.Command{
		{
			Name:        "list",
			Usage:       "list alerts",
			Description: "Shows alerts in human-readable format.",
			Action:      doAlertsList,
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:  "service, s",
					Value: &cli.StringSlice{},
					Usage: "Filters alerts by service. Multiple choices are allowed.",
				},
				&cli.StringSliceFlag{
					Name:  "host-status, S",
					Value: &cli.StringSlice{},
					Usage: "Filters alerts by status of each host. Multiple choices are allowed.",
				},
				&cli.BoolFlag{Name: "color, c", Usage: "Colorize output. default: true"},
			},
		},
		{
			Name:        "close",
			Usage:       "close alerts",
			Description: "Closes alerts. Multiple alert IDs can be specified.",
			Action:      doAlertsClose,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "reason, r", Value: "", Usage: "Reason of closing alert."},
				&cli.BoolFlag{Name: "verbose, v", Usage: "Verbose output mode"},
			},
		},
	},
}

type alertSet struct {
	Alert   *mkr.Alert
	Host    *mkr.Host
	Monitor mkr.Monitor
}

func joinMonitorsAndHosts(client *mkr.Client, alerts []*mkr.Alert) []*alertSet {
	hostsJSON, err := client.FindHosts(&mkr.FindHostsParam{
		Statuses: []string{"working", "standby", "poweroff", "maintenance"},
	})
	logger.DieIf(err)

	hosts := map[string]*mkr.Host{}
	for _, host := range hostsJSON {
		hosts[host.ID] = host
	}

	monitorsJSON, err := client.FindMonitors()
	logger.DieIf(err)

	monitors := map[string]mkr.Monitor{}
	for _, monitor := range monitorsJSON {
		monitors[monitor.MonitorID()] = monitor
	}

	alertSets := []*alertSet{}
	for _, alert := range alerts {
		alertSets = append(
			alertSets,
			&alertSet{Alert: alert, Host: hosts[alert.HostID], Monitor: monitors[alert.MonitorID]},
		)
	}
	return alertSets
}

func formatJoinedAlert(alertSet *alertSet, colorize bool) string {
	const layout = "2006-01-02 15:04:05"

	host := alertSet.Host
	monitor := alertSet.Monitor
	alert := alertSet.Alert

	hostMsg := ""
	if host != nil {
		statusMsg := host.Status
		if host.IsRetired == true {
			statusMsg = "retired"
		}
		if colorize {
			switch statusMsg {
			case "working":
				statusMsg = color.BlueString("working")
			case "standby":
				statusMsg = color.GreenString("standby")
			case "poweroff":
				statusMsg = "poweroff"
			case "maintenance":
				statusMsg = color.YellowString("maintenance")
			}
		}
		hostMsg = fmt.Sprintf(" %s %s", host.Name, statusMsg)
		roleMsgs := []string{}
		for service, roles := range host.Roles {
			roleMsgs = append(roleMsgs, fmt.Sprintf("%s:%s", service, strings.Join(roles, ",")))
		}
		hostMsg += " [" + strings.Join(roleMsgs, ", ") + "]"
	}

	monitorMsg := ""
	if monitor != nil {
		switch m := monitor.(type) {
		case *mkr.MonitorConnectivity:
			monitorMsg = ""
		case *mkr.MonitorHostMetric:
			switch alert.Status {
			case "CRITICAL":
				monitorMsg = fmt.Sprintf("%s %.2f %s %.2f", m.Metric, alert.Value, m.Operator, m.Critical)
			case "WARNING":
				monitorMsg = fmt.Sprintf("%s %.2f %s %.2f", m.Metric, alert.Value, m.Operator, m.Warning)
			default:
				monitorMsg = fmt.Sprintf("%s %.2f %s %.2f", m.Metric, alert.Value, m.Operator, m.Critical)
			}
		case *mkr.MonitorServiceMetric:
			switch alert.Status {
			case "CRITICAL":
				monitorMsg = fmt.Sprintf("%s %s %.2f %s %.2f", m.Service, m.Metric, alert.Value, m.Operator, m.Critical)
			case "WARNING":
				monitorMsg = fmt.Sprintf("%s %s %.2f %s %.2f", m.Service, m.Metric, alert.Value, m.Operator, m.Warning)
			default:
				monitorMsg = fmt.Sprintf("%s %s %.2f %s %.2f", m.Service, m.Metric, alert.Value, m.Operator, m.Critical)
			}
		case *mkr.MonitorExternalHTTP:
			statusRegexp, _ := regexp.Compile("^[2345][0-9][0-9]$")
			switch alert.Status {
			case "CRITICAL":
				if statusRegexp.MatchString(alert.Message) {
					monitorMsg = fmt.Sprintf("%s %.2f > %.2f msec, status:%s", m.URL, alert.Value, m.ResponseTimeCritical, alert.Message)
				} else {
					monitorMsg = fmt.Sprintf("%s %.2f msec, %s", m.URL, alert.Value, alert.Message)
				}
			case "WARNING":
				if statusRegexp.MatchString(alert.Message) {
					monitorMsg = fmt.Sprintf("%.2f > %.2f msec, status:%s", alert.Value, m.ResponseTimeWarning, alert.Message)
				} else {
					monitorMsg = fmt.Sprintf("%.2f msec, %s", alert.Value, alert.Message)
				}
			default:
				monitorMsg = fmt.Sprintf("%.2f > %.2f msec, status:%s", alert.Value, m.ResponseTimeCritical, alert.Message)
			}
		case *mkr.MonitorExpression:
			expression := formatExpressionOneline(m.Expression)
			switch alert.Status {
			case "CRITICAL":
				monitorMsg = fmt.Sprintf("%s %.2f %s %.2f", expression, alert.Value, m.Operator, m.Critical)
			case "WARNING":
				monitorMsg = fmt.Sprintf("%s %.2f %s %.2f", expression, alert.Value, m.Operator, m.Warning)
			case "UNKNOWN":
				monitorMsg = fmt.Sprintf("%s", expression)
			default:
				monitorMsg = fmt.Sprintf("%s %.2f", expression, alert.Value)
			}
		default:
			monitorMsg = fmt.Sprintf("%s", monitor.MonitorType())
		}
		if monitorMsg == "" {
			monitorMsg = monitor.MonitorName()
		} else {
			monitorMsg = monitor.MonitorName() + " " + monitorMsg
		}
	}
	statusMsg := alert.Status
	if colorize {
		switch alert.Status {
		case "CRITICAL":
			statusMsg = color.RedString("CRITICAL")
		case "WARNING":
			statusMsg = color.YellowString("WARNING ")
		case "UNKNOWN":
			statusMsg = "UNKNOWN "
		}
	}
	return fmt.Sprintf("%s %s %s %s%s", alert.ID, time.Unix(alert.OpenedAt, 0).Format(layout), statusMsg, monitorMsg, hostMsg)
}

var expressionNewlinePattern = regexp.MustCompile(`\s*[\r\n]+\s*`)

func formatExpressionOneline(expr string) string {
	expr = strings.Trim(expressionNewlinePattern.ReplaceAllString(expr, " "), " ")
	return strings.Replace(strings.Replace(expr, "( ", "(", -1), " )", ")", -1)
}

func doAlertsRetrieve(c *cli.Context) error {
	client := newMackerelFromContext(c)

	alerts, err := client.FindAlerts()
	logger.DieIf(err)
	PrettyPrintJSON(alerts)
	return nil
}

func doAlertsList(c *cli.Context) error {
	filterServices := c.StringSlice("service")
	filterStatuses := c.StringSlice("host-status")
	client := newMackerelFromContext(c)

	alerts, err := client.FindAlerts()
	logger.DieIf(err)
	joinedAlerts := joinMonitorsAndHosts(client, alerts)

	for _, joinAlert := range joinedAlerts {
		if len(filterServices) > 0 {
			found := false
			for _, filterService := range filterServices {
				if joinAlert.Host != nil {
					if _, ok := joinAlert.Host.Roles[filterService]; ok {
						found = true
					}
				} else {
					var service string
					if m, ok := joinAlert.Monitor.(*mkr.MonitorServiceMetric); ok {
						service = m.Service
					} else if m, ok := joinAlert.Monitor.(*mkr.MonitorExternalHTTP); ok {
						service = m.Service
					}
					found = service == filterService
				}
			}
			if !found {
				continue
			}
		}
		if len(filterStatuses) > 0 {
			found := false
			for _, filterStatus := range filterStatuses {
				if joinAlert.Host != nil && joinAlert.Host.Status == filterStatus {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		fmt.Println(formatJoinedAlert(joinAlert, c.Bool("color")))
	}
	return nil
}

func doAlertsClose(c *cli.Context) error {
	isVerbose := c.Bool("verbose")
	argAlertIDs := c.Args()
	reason := c.String("reason")

	if argAlertIDs.Len() < 1 {
		cli.ShowCommandHelp(c, "alerts")
		os.Exit(1)
	}

	client := newMackerelFromContext(c)
	for i := 0; i < argAlertIDs.Len(); i++ {
		alertID := argAlertIDs.Get(i)
		alert, err := client.CloseAlert(alertID, reason)
		logger.DieIf(err)

		logger.Log("Alert closed", alertID)
		if isVerbose == true {
			PrettyPrintJSON(alert)
		}
	}
	return nil
}
