package certprof

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanocmd/logkeys"
	"github.com/micromdm/nanocmd/workflow"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// Start starts the workflow.
func (w *Workflow) Start(ctx context.Context, step *workflow.StepStart) error {
	// load and validate the workflow context
	wfCtx, ok := step.Context.(*Context)
	if !ok {
		return workflow.ErrInvalidContext
	}
	if err := wfCtx.Validate(step.Name); err != nil {
		return fmt.Errorf("validating context: %w", err)
	}

	// assemble the CertificateList MDM command
	cmd := mdmcommands.NewCertificateListCommand(w.ider.ID())

	if !wfCtx.NoManagedOnly {
		managedOnly := true
		cmd.Command.ManagedOnly = &managedOnly
	}

	// assemble our StepEnqueuing
	se := step.NewStepEnqueueing()
	// add our command
	se.Commands = []interface{}{cmd}
	// pass along our context to the subsequent step completions
	se.Context = wfCtx

	// enqueue the step!
	return w.enq.EnqueueStep(ctx, w, se)
}

// replace replaces instances of r's map keys with their values.
func replace(in string, r map[string]string) string {
	if len(in) < 1 || r == nil || len(r) < 1 {
		return in
	}
	oldnews := make([]string, 0, len(r)*2)
	for k, v := range r {
		oldnews = append(oldnews, k, v)
	}
	repl := strings.NewReplacer(oldnews...)
	return repl.Replace(in)
}

// filter in using f.
func filter(f *Filter, in []mdmcommands.CertificateListItem) (out []mdmcommands.CertificateListItem) {
	if f == nil {
		return
	}
	for _, item := range in {
		// skip if cert is not an identity
		if !f.AllowNonIdentity && !item.IsIdentity {
			continue
		}

		// skip if no CN prefix match
		if f.CNPrefix != "" && !strings.HasPrefix(item.CommonName, f.CNPrefix) {
			continue
		}

		out = append(out, item)
	}
	return
}

// checkCriteria compares item against c to determine whether it should be replaced (re-installed).
func checkCriteria(c *Criteria, item *mdmcommands.CertificateListItem) (replace bool, reason string, err error) {
	if c == nil {
		// if there is no criteria then the mere existence of the cert
		// means we should never replace it.
		return false, "nil criteria", nil
	}

	if c.AlwaysReplace {
		return true, "always replace", nil
	}

	if c.UntilExpirySeconds > 0 {
		crt, err := x509.ParseCertificate(item.Data)
		if err != nil {
			return true, "error", fmt.Errorf("parse certificate: %w", err)
		}

		if time.Until(crt.NotAfter) < (time.Duration(c.UntilExpirySeconds) * time.Second) {
			return true, fmt.Sprintf("certificate NotAfter Until: %d; less than expiry: %d", time.Until(crt.NotAfter), c.UntilExpirySeconds), nil
		}
	}

	return false, "default false replace", nil
}

// certListStepCompleted occurs when the certificate list response is received.
func (w *Workflow) certListStepCompleted(ctx context.Context, stepResult *workflow.StepResult) error {
	// validate the correct MDM response
	if len(stepResult.CommandResults) != 1 {
		return workflow.ErrStepResultCommandLenMismatch
	}
	response, ok := stepResult.CommandResults[0].(*mdmcommands.CertificateListResponse)
	if !ok {
		return workflow.ErrIncorrectCommandType
	}
	if err := response.Validate(); err != nil {
		return fmt.Errorf("validating certificate list response: %w", err)
	}

	// validate the (carried over) workflow context
	wfCtx, ok := stepResult.Context.(*Context)
	if !ok {
		return workflow.ErrInvalidContext
	}
	if err := wfCtx.Validate(stepResult.Name); err != nil {
		return fmt.Errorf("validating context: %w", err)
	}

	logger := ctxlog.Logger(ctx, w.logger)

	// try and match the single certificate we want to replace
	cl := filter(wfCtx.Filter, response.CertificateList)

	if len(cl) > 1 {
		return fmt.Errorf("too many certs after filter: %d; before filter: %d", len(cl), len(response.CertificateList))
	} else if len(cl) == 1 {
		replace, reason, err := checkCriteria(wfCtx.Criteria, &cl[0])
		if err != nil {
			return fmt.Errorf("checking criteria: %w", err)
		}
		if !replace {
			// checkCriteria has determined this certificate need not be replaced.
			logger.Debug(logkeys.Message, fmt.Sprintf("not installing profile: %s", reason))
			// so simply exit gracefully.
			return nil
		}
		logger.Debug(logkeys.Message, fmt.Sprintf("installing profile: %s", reason))
	} else { // i.e. no certs
		logger.Debug(logkeys.Message, "no certificate found; installing profile")
	}

	profiles, err := w.store.RetrieveRawProfiles(ctx, []string{wfCtx.Profile})
	if err != nil {
		return fmt.Errorf("retrieving raw profile: %s: %w", wfCtx.Profile, err)
	}

	// replace values
	profile := replace(string(profiles[wfCtx.Profile]), wfCtx.TextReplacements)

	// assemble the MDM command
	cmd := mdmcommands.NewInstallProfileCommand(w.ider.ID())
	cmd.Command.Payload = []byte(profile)

	// create the step
	se := stepResult.NewStepEnqueueing()
	se.Name = stepNameProfile
	se.Commands = []interface{}{cmd}
	// we abandon our context here, as we're not using it for the install profile response

	return w.enq.EnqueueStep(ctx, w, se)
}

// certListStepCompleted occurs when the install profile response is received.
func (w *Workflow) profileStepCompleted(_ context.Context, stepResult *workflow.StepResult) error {
	// validate the correct MDM response
	if len(stepResult.CommandResults) != 1 {
		return workflow.ErrStepResultCommandLenMismatch
	}
	response, ok := stepResult.CommandResults[0].(*mdmcommands.InstallProfileResponse)
	if !ok {
		return workflow.ErrIncorrectCommandType
	}
	if err := response.Validate(); err != nil {
		return fmt.Errorf("validating install profile response: %w", err)
	}

	return nil
}

// StepCompleted occurs when any workflow step is completed.
func (w *Workflow) StepCompleted(ctx context.Context, stepResult *workflow.StepResult) error {
	switch stepResult.Name {
	case "":
		return w.certListStepCompleted(ctx, stepResult)
	case stepNameProfile:
		return w.profileStepCompleted(ctx, stepResult)
	default:
		return fmt.Errorf("%w: %s", workflow.ErrUnknownStepName, stepResult.Name)
	}
}
