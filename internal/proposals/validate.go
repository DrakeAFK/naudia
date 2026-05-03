package proposals

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func Validate(p *Proposal) error {
	if p == nil {
		return errors.New("proposal is nil")
	}
	if p.Type == "" {
		return errors.New("proposal type is required")
	}
	if strings.TrimSpace(p.Title) == "" {
		return errors.New("proposal title is required")
	}
	if len(p.Actions) == 0 {
		return errors.New("proposal must contain at least one action")
	}
	seen := map[string]bool{}
	for i := range p.Actions {
		a := &p.Actions[i]
		if a.ID == "" {
			a.ID = fmt.Sprintf("action-%d", i+1)
		}
		if seen[a.ID] {
			return fmt.Errorf("duplicate action id %q", a.ID)
		}
		seen[a.ID] = true
		if a.Kind == "" {
			return fmt.Errorf("action %s kind is required", a.ID)
		}
		if a.Path == "" {
			return fmt.Errorf("action %s path is required", a.ID)
		}
		if filepath.IsAbs(a.Path) || strings.Contains(filepath.Clean(a.Path), ".."+string(filepath.Separator)) {
			return fmt.Errorf("action %s path must stay inside vault", a.ID)
		}
		if (a.Kind == ActionMoveNote || a.Kind == ActionRenameNote) && a.NewPath == "" {
			return fmt.Errorf("action %s new_path is required", a.ID)
		}
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	if p.RiskLevel == "" {
		p.RiskLevel = InferRisk(p)
	}
	if p.RiskLevel == RiskHigh {
		p.RequiresConfirmation = true
	}
	return nil
}

func InferRisk(p *Proposal) RiskLevel {
	if p == nil {
		return RiskLow
	}
	if len(p.Actions) > 20 {
		return RiskHigh
	}
	risk := RiskLow
	for _, action := range p.Actions {
		switch action.Kind {
		case ActionDeleteNote, ActionMoveNote, ActionRenameNote:
			return RiskHigh
		case ActionUpdateNote, ActionRemoveLines:
			risk = RiskMedium
		}
		if strings.HasPrefix(action.Path, ".") && !strings.HasPrefix(action.Path, ".naudia/") {
			return RiskHigh
		}
		if strings.Contains(strings.ToLower(action.Path), "template") {
			risk = RiskMedium
		}
	}
	return risk
}
