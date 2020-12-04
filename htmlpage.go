package main

import (
	"fmt"
	"html/template"
	"os"
	"sort"
	"time"
)

// Project holds information on an ongoing RSO project.
type Project struct {
	Title     template.HTML // already escaped from the Reddit API
	Organizer string
	URL       string
	StartDate string // ISO 8601
	EndDate   string // ISO 8601

	Registers             []string // sorted nicely
	InstrumentsByRegister map[string][]Instrument

	LastUpdateDate      string
	LastUpdatePermalink string
}

// Video holds information on a YouTube video.
type Video struct {
	Title string
	ID    string
}

// ProjectsByEndDate implements sort.Interface for soring by EndDate.
type ProjectsByEndDate []Project

func (a ProjectsByEndDate) Len() int           { return len(a) }
func (a ProjectsByEndDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ProjectsByEndDate) Less(i, j int) bool { return a[i].EndDate < a[j].EndDate }

func instrumentsByRegister(instruments []Instrument) map[string][]Instrument {
	m := make(map[string][]Instrument)
	for _, instr := range instruments {
		m[instr.Register] = append(m[instr.Register], instr)
	}
	return m
}

func createHTMLPage(client *DataClient) {
	var projects []Project

	for _, post := range client.Posts {
		if !isProject(&post) {
			continue
		}
		deadline := findDeadline(post.SelfText, int64(post.CreatedUTC))
		// Filter out finished projects.
		if time.Now().After(deadline) {
			continue
		}
		byreg := instrumentsByRegister(findInstruments(post.SelfText))
		var registers []string
		for _, reg := range Registers {
			if _, ok := byreg[reg]; ok {
				registers = append(registers, reg)
			}
		}
		p := Project{
			Title:                 template.HTML(post.Title),
			Organizer:             post.Author,
			URL:                   post.URL,
			StartDate:             time.Unix(int64(post.CreatedUTC), 0).Format("2006-01-02"),
			EndDate:               deadline.Format("2006-01-02"),
			Registers:             registers,
			InstrumentsByRegister: byreg,
		}
		if lastUpdate := findUpdateComment(&post, client.WeeklyUpdates); lastUpdate != nil {
			p.LastUpdatePermalink = lastUpdate.Permalink
			ts := lastUpdate.CreatedUTC
			if lastUpdate.Edited > 0 {
				ts = lastUpdate.Edited
			}
			diff := time.Now().Sub(time.Unix(int64(ts), 0)).Hours() / 24
			if diff < 1 {
				p.LastUpdateDate = "today"
			} else if diff < 2 {
				p.LastUpdateDate = "yesterday"
			} else {
				p.LastUpdateDate = fmt.Sprintf("%d days ago", int(diff))
			}
		}
		projects = append(projects, p)
	}

	sort.Sort(ProjectsByEndDate(projects))

	v := &client.Videos[len(client.Videos)-1]
	latestVideo := Video{
		Title: v.Title,
		ID:    v.ResourceId.VideoId,
	}

	tmpl, err := template.ParseFiles("template.html")
	if err != nil {
		fmt.Println(err)
		return
	}

	f, err := os.Create("static/index.html")
	if err != nil {
		fmt.Println(err)
		return
	}

	err = tmpl.Execute(f, map[string]interface{}{
		"Projects":    projects,
		"LatestVideo": latestVideo,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
}
