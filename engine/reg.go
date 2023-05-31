package engine

import "github.com/micromdm/nanocmd/workflow"

func in(s []string, i string) int {
	for j, v := range s {
		if v == i {
			return j
		}
	}
	return -1
}

func (e *Engine) registerAllResp(name string, allRespReqTypes []string) {
	for _, reqType := range allRespReqTypes {
		regNames := e.allResps[reqType]
		if in(regNames, name) == -1 {
			regNames = append(regNames, name)
			e.allResps[reqType] = regNames
		}
	}
}

// RegisterWorkflow associates w with the engine by name.
func (e *Engine) RegisterWorkflow(w workflow.Workflow) error {
	e.workflowsMu.Lock()
	defer e.workflowsMu.Unlock()
	e.workflows[w.Name()] = w
	if cfg := w.Config(); cfg != nil {
		e.registerAllResp(w.Name(), cfg.AllCommandResponseRequestTypes)
	}
	e.logger.Debug("msg", "registered workflow", "name", w.Name())
	return nil
}

func (e *Engine) unregisterAllResp(name string) {
	for k, v := range e.allResps {
		pos := in(v, name)
		if pos >= 0 && len(v) > pos {
			e.allResps[k] = append(v[0:pos], v[pos+1:]...)
		}
	}
}

func (e *Engine) allRespWorkflows(reqType string) (workflows []workflow.Workflow) {
	e.workflowsMu.RLock()
	defer e.workflowsMu.RUnlock()
	for _, name := range e.allResps[reqType] {
		w := e.Workflow(name)
		if w != nil {
			workflows = append(workflows, w)
		}
	}
	return
}

// UnregisterWorkflow dissociates the named workflow from the engine by name.
func (e *Engine) UnregisterWorkflow(name string) error {
	e.workflowsMu.Lock()
	defer e.workflowsMu.Unlock()
	if _, ok := e.workflows[name]; ok {
		delete(e.workflows, name)
		e.unregisterAllResp(name)
		e.logger.Debug("msg", "unregistered workflow", "name", name)
	} else {
		e.logger.Info(
			"msg", "unregistered workflow",
			"name", name,
			"err", "workflow name not found",
		)
	}
	return nil
}

// Workflow returns the registered workflow by name.
func (e *Engine) Workflow(name string) workflow.Workflow {
	e.workflowsMu.RLock()
	defer e.workflowsMu.RUnlock()
	return e.workflows[name]
}

// WorkflowRegistered returns true if the workflow name is registered.
func (e *Engine) WorkflowRegistered(name string) bool {
	e.workflowsMu.RLock()
	defer e.workflowsMu.RUnlock()
	_, ok := e.workflows[name]
	return ok
}

// eventWorkflows returns workflows that are configured to receive the ev event.
func (e *Engine) eventWorkflows(ev workflow.EventFlag) (workflows []workflow.Workflow) {
	e.workflowsMu.RLock()
	defer e.workflowsMu.RUnlock()
	for _, w := range e.workflows {
		if cfg := w.Config(); cfg != nil && ev&cfg.Events > 0 {
			// w "subscribes" to ev; add w to the list
			workflows = append(workflows, w)
		}
	}
	return
}
