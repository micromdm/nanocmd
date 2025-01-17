package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/workflow"
	"github.com/micromdm/plist"
)

type newContextValuer interface {
	NewContextValue(string) workflow.ContextMarshaler
}

// workflowStepStartFromEngine creates a new workflow start step from raw engine input.
func workflowStepStartFromEngine(instanceID string, newCtx newContextValuer, rawContext []byte, ids []string, e *workflow.Event, mCtx *workflow.MDMContext) (*workflow.StepStart, error) {
	ss := &workflow.StepStart{
		StepContext: workflow.StepContext{
			InstanceID: instanceID,
			Name:       "", // workflow step name is blank when starting
			Context:    newCtx.NewContextValue(""),
		},
		IDs:   ids,
		Event: e,
	}
	if mCtx != nil {
		ss.MDMContext = *mCtx
	}
	if ss.Context != nil && len(rawContext) > 0 {
		if err := ss.Context.UnmarshalBinary(rawContext); err != nil {
			return ss, fmt.Errorf("unmarshal context: %w", err)
		}
	}
	return ss, nil
}

// storageCommandRawFromWorkflowCommand converts a workflow command to a storage command.
func storageCommandRawFromWorkflowCommand(cmd interface{}) (*storage.StepCommandRaw, error) {
	if cmd == nil {
		return nil, errors.New("invalid command (nil)")
	}
	// make sure we're able to extract the generic set of command attributes
	genCmder, ok := cmd.(mdmcommands.GenericCommander)
	if !ok {
		return nil, errors.New("invalid command type")
	}
	// extract them
	genCmd := genCmder.GenericCommand()
	if genCmd == nil {
		return nil, errors.New("invalid command (nil generic command)")
	}
	// some sanity tests
	if genCmd.CommandUUID == "" {
		return nil, errors.New("empty command uuid")
	}
	if genCmd.Command.RequestType == "" {
		return nil, errors.New("empty request type")
	}
	// marshal into plist XML
	rawCmd, err := plist.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("marshal command: %w", err)
	}
	// construct our storage step command
	stepCmd := &storage.StepCommandRaw{
		CommandUUID: genCmd.CommandUUID,
		RequestType: genCmd.Command.RequestType,
		Command:     rawCmd,
	}
	return stepCmd, nil
}

// storageStepEnqueuingWithConfigFromWorkflowStepEnqueueing converts a workflow step (for enqueueing) to a storage step.
func storageStepEnqueuingWithConfigFromWorkflowStepEnqueueing(n workflow.Namer, defaultTimeout time.Time, se *workflow.StepEnqueueing) (*storage.StepEnqueuingWithConfig, error) {
	ss := &storage.StepEnqueuingWithConfig{
		StepEnqueueing: storage.StepEnqueueing{
			StepContext: storage.StepContext{
				InstanceID:   se.InstanceID,
				Name:         se.Name,
				WorkflowName: n.Name(),
			},
			IDs: se.IDs,
		},
		Timeout:  se.Timeout,
		NotUntil: se.NotUntil,
	}
	if ss.Timeout.IsZero() {
		ss.Timeout = defaultTimeout
	}
	if se.Context != nil {
		var err error
		ss.Context, err = se.Context.MarshalBinary()
		if err != nil {
			return ss, fmt.Errorf("marshal context: %w", err)
		}
	}
	for _, cmd := range se.Commands {
		sc, err := storageCommandRawFromWorkflowCommand(cmd)
		if err != nil {
			// TODO: more error context? (rt, uuid)
			return ss, fmt.Errorf("converting command: %w", err)
		}
		ss.Commands = append(ss.Commands, *sc)
	}
	return ss, nil
}

// storageStepCommandFromRawResponse converts a raw response into a StepCommandResult.
func storageStepCommandFromRawResponse(reqType string, rawResp []byte) (*storage.StepCommandResult, interface{}, error) {
	response, err := workflowCommandResponseFromRawResponse(reqType, rawResp)
	if err != nil {
		return nil, response, fmt.Errorf("converting response: %w", err)
	}
	genResper, ok := response.(mdmcommands.GenericResponser)
	if !ok {
		// this would be odd as the response comes from the mdmcommands module
		return nil, response, errors.New("invalid response type")
	}
	genResp := genResper.GetGenericResponse()
	if genResp == nil {
		return nil, response, errors.New("invalid response (nil generic response)")
	}
	sc := &storage.StepCommandResult{
		CommandUUID:  genResp.CommandUUID,
		RequestType:  reqType,
		ResultReport: rawResp,
		Completed:    genResp.Status != "" && genResp.Status != "NotNow",
	}
	return sc, response, sc.Validate()
}

// workflowCommandResponseFromRawResponse converts a raw XML plist of a command response to a workflow response.
func workflowCommandResponseFromRawResponse(reqType string, rawResp []byte) (interface{}, error) {
	resp := mdmcommands.NewResponse(reqType)
	if resp == nil {
		return nil, fmt.Errorf("no response for request type: %s", reqType)
	}
	err := plist.Unmarshal(rawResp, resp)
	if err != nil {
		return resp, fmt.Errorf("unmarshal response: %w", err)
	}
	return resp, nil
}

// workflowStepResultFromStorageStep converts a storage step into a workflow step result.
// As a special optimization we accept a uuid and resp to short-circuit an already-
// unmarshalled workflow step command to be used.
// If ignoreEmptyResp is set then we do not require completed commands and skip
// parsing empty responses.
func workflowStepResultFromStorageStepResult(ss *storage.StepResult, newCtx newContextValuer, ignoreEmptyResp bool, uuid string, spResp interface{}) (*workflow.StepResult, error) {
	if len(ss.IDs) != 1 {
		// results (MDM command responses) can only be for a single ID
		return nil, errors.New("incorrect id count")
	}
	sr := &workflow.StepResult{
		StepContext: workflow.StepContext{
			InstanceID: ss.InstanceID,
			Name:       ss.Name,
			Context:    newCtx.NewContextValue(ss.Name),
		},
		ID: ss.IDs[0],
	}
	if sr.Context != nil && len(ss.Context) > 0 {
		if err := sr.Context.UnmarshalBinary(ss.Context); err != nil {
			return sr, fmt.Errorf("unmarshal context: %w", err)
		}
	}
	for _, cmd := range ss.Commands {
		if cmd.CommandUUID == uuid {
			sr.CommandResults = append(sr.CommandResults, spResp)
			continue
		}
		if ignoreEmptyResp && len(cmd.ResultReport) < 1 {
			continue
		}
		resp, err := workflowCommandResponseFromRawResponse(cmd.RequestType, cmd.ResultReport)
		if err != nil {
			// TODO: more error context? (rt, uuid)
			return sr, fmt.Errorf("converting response: %w", err)
		}
		sr.CommandResults = append(sr.CommandResults, resp)
	}
	return sr, nil
}
