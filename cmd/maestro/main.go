package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maestro/maestro.go/internal/application"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	var (
		command      string
		workflowFile string
		inputJSON    string
		port         int
		debug        bool
		trace        bool
	)

	flag.StringVar(&workflowFile, "workflow", "", "Path to workflow YAML file")
	flag.StringVar(&workflowFile, "f", "", "Path to workflow YAML file (shorthand)")
	flag.StringVar(&inputJSON, "input", "{}", "Input data as JSON")
	flag.StringVar(&inputJSON, "i", "{}", "Input data as JSON (shorthand)")
	flag.IntVar(&port, "port", 8080, "Port to listen on (for serve command)")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.BoolVar(&trace, "trace", false, "Enable trace logging")
	flag.Parse()

	logLevel := zerolog.InfoLevel
	if debug {
		logLevel = zerolog.DebugLevel
	}
	if trace {
		logLevel = zerolog.TraceLevel
	}

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger().Level(logLevel)

	if flag.NArg() < 1 {
		printUsage()
		os.Exit(1)
	}

	command = flag.Arg(0)

	switch command {
	case "execute":
		if flag.NArg() >= 2 {
			workflowFile = flag.Arg(1)
		} else if workflowFile == "" {
			fmt.Println("Error: workflow file required for execute command")
			printUsage()
			os.Exit(1)
		}
		executeWorkflow(workflowFile, inputJSON)

	case "serve":
		serveOrchestrator(port)

	case "validate":
		if flag.NArg() >= 2 {
			workflowFile = flag.Arg(1)
		} else if workflowFile == "" {
			fmt.Println("Error: workflow file required for validate command")
			printUsage()
			os.Exit(1)
		}
		validateWorkflow(workflowFile)

	case "help":
		printUsage()

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Maestro - Polyglot API Orchestrator

Usage:
  maestro <command> [options]

Commands:
  execute <workflow.yaml>  Execute a workflow
  serve                    Start the orchestrator server
  validate <workflow.yaml> Validate a workflow file
  help                     Show this help message

Options:
  -f, --workflow   Path to workflow YAML file
  -i, --input      Input data as JSON (default: {})
  --port           Port to listen on for serve command (default: 8080)
  --debug          Enable debug logging
  --trace          Enable trace logging

Examples:
  maestro execute user_onboarding.yaml --input '{"email":"user@example.com"}'
  maestro serve --port 8080
  maestro validate workflows/order_processing.yaml`)
}

func executeWorkflow(workflowFile, inputJSON string) {
	logger := log.With().Str("command", "execute").Logger()
	logger.Info().Str("workflow", workflowFile).Msg("Executing workflow")

	var input map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		logger.Fatal().Err(err).Msg("Failed to parse input JSON")
	}

	orch := application.New(logger)

	if err := orch.LoadWorkflow(workflowFile); err != nil {
		logger.Fatal().Err(err).Msg("Failed to load workflow")
	}

	workflows := orch.ListWorkflows()
	if len(workflows) == 0 {
		logger.Fatal().Msg("No workflows loaded")
	}
	workflowName := workflows[0]

	logger.Info().
		Str("workflow", workflowName).
		Msg("Workflow loaded successfully")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info().Msg("Received interrupt signal, cancelling workflow")
		cancel()
	}()

	result, err := orch.ExecuteWorkflow(ctx, workflowName, input)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Workflow execution failed")
		os.Exit(1)
	}

	logger.Info().
		Str("workflow_id", result.WorkflowID).
		Str("status", result.Status.String()).
		Dur("duration", result.CompletedAt.Sub(result.StartedAt)).
		Interface("output", result.Output).
		Msg("Workflow completed")

	if outputJSON, err := json.MarshalIndent(result.Output, "", "  "); err == nil {
		fmt.Println("\nOutput:")
		fmt.Println(string(outputJSON))
	}
}

func serveOrchestrator(port int) {
	logger := log.With().Str("command", "serve").Logger()
	logger.Info().Int("port", port).Msg("Starting orchestrator server")

	fmt.Printf("\n Maestro Orchestrator Server\n")
	fmt.Printf("   Listening on port %d\n", port)
	fmt.Printf("   Press Ctrl+C to stop\n\n")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info().Msg("Shutting down orchestrator server")
}

func validateWorkflow(workflowFile string) {
	logger := log.With().Str("command", "validate").Logger()
	logger.Info().Str("workflow", workflowFile).Msg("Validating workflow")

	orch := application.New(logger)

	if err := orch.LoadWorkflow(workflowFile); err != nil {
		logger.Error().Err(err).Msg("Workflow validation failed")
		os.Exit(1)
	}

	logger.Info().Msg("✅ Workflow is valid")
	fmt.Println("✅ Workflow validation successful")
}
