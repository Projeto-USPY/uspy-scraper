package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/Projeto-USPY/uspy-backend/config"
	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-scraper/collector"
)

type scrapingTargets []string

func (sT *scrapingTargets) String() string {
	return fmt.Sprint(*sT)
}

func (sT *scrapingTargets) Set(value string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	rgx := regexp.MustCompile("[0-9a-zA-Z]+(,[0-9a-zA-Z]+)*")
	*sT = rgx.FindAllString(value, -1)

	return err
}

var (
	mode    string
	method  string
	targets scrapingTargets
)

func init() {
	flag.StringVar(&mode, "mode", "dev", "Set this to prod or dev to determine which environment to output scraped data.")
	flag.StringVar(&method, "method", "log", "Set this to build, update or log to determine if you are building or updating the database.")
	flag.Var(&targets, "targets", "Set this to a comma-separated list of targets for scraping (see collector.go for options)")
}

func main() {
	flag.Parse()

	config.Setup()
	env := db.SetupDB()

	if len(targets) == 0 {
		log.Println("no targets specified, defaulting to \"subjects, offerings\"")
		targets = []string{"subjects", "offerings"}
	}

	log.Printf("Running %s in %s\n", method, mode)

	for _, name := range targets {
		col, ok := collector.Collectors[name]

		if !ok {
			log.Println(name, "is an invalid target")
			continue
		}

		log.Println("executing", name)
		if objs, err := col.Collect(env); err == nil {
			if method == "update" {
				if updErr := collector.Update(env, col, objs); updErr != nil {
					log.Fatalln("error updating objects:", updErr)
				}
			} else if method == "build" {
				if mode == "prod" && os.Getenv("LOCAL") != "FALSE" {
					log.Fatal("You cannot build the production database locally. This can only be run from inside GCP.")
				}

				if buildErr := collector.Build(env, col, objs); buildErr != nil {
					log.Fatalln("error building objects:", buildErr)
				}
			} else if method == "log" {
				log.Println(objs)
			} else {
				log.Fatalln("method", method, "is not known, please use build/update")
			}
		} else {
			log.Println("error collecting data:", err)
		}
	}
}
