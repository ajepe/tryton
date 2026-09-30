package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ajepe/talon/conformance"
)

func main() {
	var (
		pythonURL   = flag.String("python", "http://localhost:8000", "Python trytond URL")
		goURL       = flag.String("go", "http://localhost:8001", "Go server URL")
		dbName      = flag.String("db", "test", "database name")
		casesDir    = flag.String("cases", "./cases", "test cases directory")
		fixturesDir = flag.String("fixtures", "./fixtures", "fixtures directory")
		login       = flag.String("login", "", "login used to authenticate both servers")
		password    = flag.String("password", "", "password used to authenticate both servers")
		report      = flag.String("report", "", "report output file")
		timeout     = flag.Duration("timeout", 0, "per-call timeout")
	)
	flag.Parse()

	runner := conformance.NewRunner(*pythonURL, *goURL, *dbName)
	if *timeout > 0 {
		runner.SetTimeout(*timeout)
	}

	ctx := context.Background()
	if *login != "" {
		if err := runner.Authenticate(ctx, *login, *password); err != nil {
			log.Fatalf("authentication: %v", err)
		}
	}

	fixtures, err := conformance.LoadFixtures(*fixturesDir)
	if err != nil {
		log.Fatalf("loading fixtures: %v", err)
	}
	if err := runner.ApplyFixtures(ctx, fixtures); err != nil {
		log.Fatalf("applying fixtures: %v", err)
	}

	cases, err := conformance.LoadCases(*casesDir)
	if err != nil {
		log.Fatalf("loading cases: %v", err)
	}

	if err := runner.Run(ctx, cases); err != nil {
		log.Fatalf("running cases: %v", err)
	}

	fmt.Println(runner.Report())

	if *report != "" {
		if err := runner.WriteReport(*report); err != nil {
			log.Fatalf("writing report: %v", err)
		}
	}

	if runner.Failures() > 0 {
		os.Exit(1)
	}
}
