package main

import (
	"context"
	"flag"
	"log"
	"os"
	"sort"
	"time"

	"github.com/joho/godotenv"

	"github.com/bobheadxi/toggl-to-clockify/clockify"
	"github.com/bobheadxi/toggl-to-clockify/toggl"
)

var (
	help  = flag.Bool("help", false, "display help text")
	start = flag.Int("start", 7, "start time (days ago)")
	end   = flag.Int("end", 0, "end time (days ago")
	exec  = flag.Bool("exec", false, "execute updates instead of just printing them")

	togglProjectName = flag.String("toggl.project", "", "toggle project to sync")

	clockifyWorkspace     = flag.String("clockify.workspace", "", "clockify workspace 'clockify.project' belongs to")
	clockifyProjectName   = flag.String("clockify.project", "", "clockify project to sync to")
	clockifyForceBillable = flag.Bool("clockify.billable", false, "force billable status")
)

func main() {
	godotenv.Load()
	flag.Parse()
	if *help {
		println("Toggl to Clockify exporter.")
		println("\nUSAGE:\n  toggl-to-clockify -toggl.project=\"sumus-portal\" -clockify.workspace=\"bobheadxi\" -clockify.project=\"Sumus Portal\"")
		println("\nFLAGS:")
		flag.PrintDefaults()
		return
	}

	var (
		ctx           = context.Background()
		togglUser     = os.Getenv("TOGGL_USER")
		togglToken    = os.Getenv("TOGGL_TOKEN")
		clockifyToken = os.Getenv("CLOCKIFY_TOKEN")
		startTime     = time.Now().Add(-(time.Duration(*start) * 24 * time.Hour))
		endTime       = time.Now().Add(-(time.Duration(*end) * 24 * time.Hour))
	)

	log.Printf("seeking entries between %s and %s", startTime, endTime)

	// get toggl data
	togglClient := toggl.New(togglUser, togglToken)
	togglProject, err := togglClient.GetProject(ctx, *togglProjectName)
	if err != nil {
		log.Fatalf("failed to get project: %v", err)
	}
	log.Printf("toggl: project '%s' (%d)", togglProject.Name, togglProject.ID)

	// retrieve entries to export
	togglEntries, err := togglClient.GetEntries(ctx,
		startTime, endTime,
		togglProject.ID)
	if err != nil {
		log.Fatalf("failed to get toggl entries: %v", err)
	}
	sort.Sort(togglEntries)

	// get clockify data
	clockifyClient := clockify.New(clockifyToken)
	clockifyProject, err := clockifyClient.FindProject(ctx, *clockifyWorkspace, *clockifyProjectName)
	if err != nil {
		log.Fatalf("failed to find clockify project: %v", err)
	}
	log.Printf("clockify: project '%s' (%s), workspace '%s' (%s)",
		clockifyProject.Name, clockifyProject.ID, *clockifyWorkspace, clockifyProject.WorkspaceID)

	// generate entries to import
	var count int
	for _, entry := range togglEntries {
		billable := entry.Billable
		if *clockifyForceBillable {
			billable = true
		}
		generated := clockify.Entry{
			Start:       entry.Start,
			End:         entry.Stop,
			Description: entry.Description,
			ProjectID:   clockifyProject.ID,

			Billable: billable,
			// Tasks
			// Tags
		}
		count++
		if *exec {
			log.Printf("creating entry '%s' (%s ~ %s)",
				generated.Description, generated.Start.String(), generated.End.String())
			if err := clockifyClient.AddEntry(ctx, clockifyProject.WorkspaceID, generated); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Printf("generated entry: %+v", generated)
		}
	}
	if *exec {
		log.Printf("exported %d entries", count)
	} else {
		log.Printf("%d entries to export", count)
	}
}
