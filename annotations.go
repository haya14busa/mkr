package main

import (
	"os"

	mkr "github.com/mackerelio/mackerel-client-go"
	"github.com/mackerelio/mkr/logger"
	"gopkg.in/urfave/cli.v2"
)

var commandAnnotations = &cli.Command{
	Name: "annotations",
	Description: `
    Manipulate graph annotations. Requests APIs under "/api/v0/graph-annotations".
    See https://mackerel.io/api-docs/entry/graph-annotations .
`,
	Subcommands: []*cli.Command{
		{
			Name:        "create",
			Usage:       "create a graph annotation",
			Description: "Creates a graph annotation.",
			Action:      doAnnotationsCreate,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "title", Usage: "Title for annotation"},
				&cli.StringFlag{Name: "description", Usage: "Description for annotation"},
				&cli.IntFlag{Name: "from", Usage: "Starting time (epoch seconds)"},
				&cli.IntFlag{Name: "to", Usage: "Ending time (epoch seconds)"},
				&cli.StringFlag{Name: "service, s", Usage: "Service name for annotation"},
				&cli.StringSliceFlag{
					Name:  "role, r",
					Value: &cli.StringSlice{},
					Usage: "Roles for annotation. Multiple choices are allowed",
				},
			},
		},
		{
			Name:        "list",
			Usage:       "list annotations",
			Description: "Shows annotations by service name and duration(from and to)",
			Action:      doAnnotationsList,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "service, s", Usage: "Service name for annotation"},
				&cli.IntFlag{Name: "from", Usage: "Starting time (epoch seconds)"},
				&cli.IntFlag{Name: "to", Usage: "Ending time (epoch seconds)"},
			},
		},
		{
			Name:        "update",
			Usage:       "update annotation",
			Description: "Updates an annotation",
			Action:      doAnnotationsUpdate,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: "Annotation ID."},
				&cli.StringFlag{Name: "service, s", Usage: "Service name for annotation"},
				&cli.StringFlag{Name: "title", Usage: "Title for annotation"},
				&cli.StringFlag{Name: "description", Usage: "Description for annotation"},
				&cli.IntFlag{Name: "from", Usage: "Starting time (epoch seconds)"},
				&cli.IntFlag{Name: "to", Usage: "Ending time (epoch seconds)"},
				&cli.StringSliceFlag{
					Name:  "role, r",
					Value: &cli.StringSlice{},
					Usage: "Roles for annotation. Multiple choices are allowed",
				},
			},
		},
		{
			Name:        "delete",
			Usage:       "delete annotation",
			Description: "Delete graph annotation by annotation id",
			Action:      doAnnotationsDelete,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: "Graph annotation ID"},
			},
		},
	},
}

func doAnnotationsCreate(c *cli.Context) error {
	title := c.String("title")
	description := c.String("description")
	from := c.Int64("from")
	to := c.Int64("to")
	service := c.String("service")
	roles := c.StringSlice("role")

	if service == "" || from == 0 || to == 0 {
		_ = cli.ShowCommandHelp(c, "create")
		os.Exit(1)
	}

	client := newMackerelFromContext(c)
	_, err := client.CreateGraphAnnotation(&mkr.GraphAnnotation{
		Title:       title,
		Description: description,
		From:        from,
		To:          to,
		Service:     service,
		Roles:       roles,
	})
	logger.DieIf(err)
	return nil
}

func doAnnotationsList(c *cli.Context) error {
	service := c.String("service")
	from := c.Int64("from")
	to := c.Int64("to")

	if service == "" || from == 0 || to == 0 {
		_ = cli.ShowCommandHelp(c, "list")
		os.Exit(1)
	}

	client := newMackerelFromContext(c)
	annotations, err := client.FindGraphAnnotations(service, from, to)
	logger.DieIf(err)
	PrettyPrintJSON(annotations)
	return nil
}

func doAnnotationsUpdate(c *cli.Context) error {
	annotationID := c.String("id")
	title := c.String("title")
	description := c.String("description")
	from := c.Int64("from")
	to := c.Int64("to")
	service := c.String("service")
	roles := c.StringSlice("role")

	if service == "" || from == 0 || to == 0 {
		_ = cli.ShowCommandHelp(c, "update")
		os.Exit(1)
	}

	client := newMackerelFromContext(c)
	annotation, err := client.UpdateGraphAnnotation(annotationID, &mkr.GraphAnnotation{
		Title:       title,
		Description: description,
		From:        from,
		To:          to,
		Service:     service,
		Roles:       roles,
	})
	logger.DieIf(err)
	PrettyPrintJSON(annotation)
	return nil
}

func doAnnotationsDelete(c *cli.Context) error {
	annotationID := c.String("id")

	if annotationID == "" {
		_ = cli.ShowCommandHelp(c, "delete")
		os.Exit(1)
	}

	client := newMackerelFromContext(c)
	annotation, err := client.DeleteGraphAnnotation(annotationID)
	logger.DieIf(err)
	PrettyPrintJSON(annotation)
	return nil
}
