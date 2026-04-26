package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/runtime"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to runtime config yaml")
	input := flag.String("input", "", "user input")
	dryRun := flag.Bool("dry-run", false, "load config and skills, run skill search, do not call model")
	listSkills := flag.Bool("list-skills", false, "list loaded skill metadata")
	debug := flag.Bool("debug", false, "write detailed chat request/error traces into the run directory")
	flag.Parse()

	if *input == "" && !*listSkills {
		fmt.Fprintln(os.Stderr, "missing -input")
		os.Exit(2)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	rt, err := runtime.New(cfg)
	if err != nil {
		log.Fatalf("init runtime: %v", err)
	}
	rt.Debug = *debug

	if *listSkills {
		for _, s := range rt.Registry.List() {
			fmt.Printf("- %s | %s | %s\n", s.ID, s.Name, s.Description)
		}
		return
	}

	if *dryRun {
		if err := rt.DryRun(*input); err != nil {
			log.Fatalf("dry-run: %v", err)
		}
		fmt.Println(rt.Store.PrintSummary())
		fmt.Println("dry-run completed: see dry-run/skill-search.json")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Model.TimeoutSeconds+30)*time.Second)
	defer cancel()
	res, err := rt.Run(ctx, *input)
	if err != nil {
		if *debug {
			fmt.Fprintln(os.Stderr, "run_id:", rt.Store.RunID)
			fmt.Fprintln(os.Stderr, "run_dir:", rt.Store.Root)
		}
		log.Fatalf("run: %v", err)
	}
	fmt.Println(res.FinalText)
	fmt.Println("\n---")
	fmt.Println("run_id:", res.RunID)
	fmt.Println("run_dir:", res.RunDir)
}
