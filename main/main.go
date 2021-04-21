package main

import (
	"flag"
	"log"
	"os"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-scraper/manager"
)

var (
	mode   string
	method string
)

func init() {
	flag.StringVar(&mode, "mode", "dev", "Set this to prod/dev to determine which environment to output scraped data.")
	flag.StringVar(&method, "method", "update", "Set this to build to determine if you are building or updating the database.")
}

func main() {
	flag.Parse()
	env := db.SetupDB()

	log.Printf("Running %s in %s\n", method, mode)
	for name, obj := range manager.Managers {
		log.Println("executing", name)
		if objs, err := obj.Collect(); err == nil {
			if method == "update" {
				if updErr := obj.Update(env, objs); updErr != nil {
					log.Fatalln("error updating objects:", updErr)
				}
			} else if method == "build" {
				if mode == "prod" && os.Getenv("LOCAL") != "FALSE" {
					log.Fatal("You cannot build the production database locally. This can only be run from inside GCP.")
				}

				if buildErr := obj.Build(env, objs); buildErr != nil {
					log.Fatalln("error building objects:", buildErr)
				}
			} else {
				log.Fatalln("method", method, "is not known, please use build/update")
			}
		} else {
			log.Println("error collecting data:", err)
		}
	}
}
