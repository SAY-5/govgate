package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/SAY-5/govgate/apps/register/internal/api"
	"github.com/SAY-5/govgate/apps/register/internal/checklist"
	"github.com/SAY-5/govgate/apps/register/internal/register"
	"github.com/SAY-5/govgate/apps/register/internal/scoring"
)

// assess scores a submission JSON against a checklist and prints a register
// entry (without persisting it). It is the offline counterpart to the HTTP
// submit endpoint and is used to produce golden reports.
func assess(args []string) error {
	fs := flag.NewFlagSet("assess", flag.ContinueOnError)
	checklistPath := fs.String("checklist", "checklists/default.yaml", "checklist YAML path")
	input := fs.String("input", "-", "submission JSON file, or - for stdin")
	if err := fs.Parse(args); err != nil {
		return err
	}

	c, err := checklist.LoadFile(*checklistPath)
	if err != nil {
		return err
	}

	raw, err := readInput(*input)
	if err != nil {
		return err
	}
	var in api.SubmitInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return fmt.Errorf("parse submission: %w", err)
	}

	assessment := scoring.Score(c, in.Submission, in.Judgments)
	entry := register.Entry{
		Submission:       in.Submission,
		ChecklistName:    c.Name,
		ChecklistVersion: c.Version,
		Assessment:       assessment,
		Status:           register.StatusPending,
	}
	out, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func readInput(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}
