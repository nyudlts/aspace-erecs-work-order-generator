package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/nyudlts/go-aspace"
	"os"
	"regexp"
	"strings"
)

var (
	repoID     int
	resourceID int
	client     *aspace.ASClient
	config     string
	env        string
	timeout    int
	err        error
	cuid       bool
)

var repoMap = map[int]string{2: "tamwag", 3: "fales", 6: "archives"}

func init() {
	flag.IntVar(&repoID, "repo-id", 0, "")
	flag.IntVar(&resourceID, "resource-id", 0, "")
	flag.IntVar(&timeout, "timeout", 20, "")
	flag.StringVar(&config, "config", "", "")
	flag.StringVar(&env, "env", "", "")
	flag.BoolVar(&cuid, "cuid", false, "")
}

func main() {
	flag.Parse()
	if repoID == 0 || resourceID == 0 {
		panic(fmt.Errorf("Either repo-id or resource-id not set"))
	}

	repoCode := repoMap[repoID]
	//check that the repo code was valid

	//get a client
	client, err = aspace.NewClient(config, env, timeout)
	if err != nil {
		panic(err.Error())
	}

	//get the resource
	resource, err := client.GetResource(repoID, resourceID)
	if err != nil {
		panic(err)
	}

	//convert eadid into resourceIdentifier in work order format
	resID := strings.ReplaceAll(resource.EADID, "_", ".")
	resID = strings.ToUpper(resID)
	resourceCall := strings.ReplaceAll(resID, ".", "")
	resourceCall = strings.ToLower(resourceCall)

	//create the output file and writer
	workOrderFilename := fmt.Sprintf("%s_%s_aspace_wo.tsv", repoCode, resourceCall)
	outFile, err := os.Create(workOrderFilename)
	if err != nil {
		panic(err.Error())
	}
	defer outFile.Close()
	writer := bufio.NewWriter(outFile)
	writer.WriteString("Resource ID\tRef ID\tURI\tContainer Indicator 1\tContainer Indicator 2\tContainer Indicator 3\tTitle\tComponent ID\n")
	writer.Flush()

	//get a list of digital object IDs for a collection
	doURLs, err := client.GetDigitalObjectIDsForResource(repoID, resourceID)
	if err != nil {
		panic(err.Error())
	}

	//loop through the DOURLs
	for _, doURL := range doURLs {
		_, doID, err := aspace.URISplit(doURL)
		if err != nil {
			panic(err)
		}

		//request the digital object
		do, err := client.GetDigitalObject(repoID, doID)
		if err != nil {
			panic(err.Error())
		}

		erPtn := regexp.MustCompile("^[FA|TW|UA]\\w*\\d{1,4}\\wER\\w\\d{1,4}")
		//check that the digital object id is an electronic record component
		if erPtn.MatchString(do.DigitalObjectID) {
			//get the parent AO url
			aoURL := do.LinkedInstances[0].Ref
			_, aoID, err := aspace.URISplit(aoURL)
			if err != nil {
				panic(err)
			}

			//get the parent archival object
			ao, err := client.GetArchivalObject(repoID, aoID)
			if err != nil {
				panic(err.Error())
			}

			var componentID string
			if cuid {
				componentID = ao.ComponentId
			} else {
				componentID = ""
			}

			msg := fmt.Sprintf("%s\t%s\t%s\t\t%s\t\t%s\t%s\n", resID, ao.RefID, ao.URI, do.DigitalObjectID, ao.Title, componentID)
			writer.WriteString(msg)
			writer.Flush()
		}
	}
}
