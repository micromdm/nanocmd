package test

import (
	"context"
	"fmt"
	"os"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanocmd/workflow"
	"github.com/micromdm/plist"
)

type MDMCommandEventer interface {
	MDMCommandResponseEvent(ctx context.Context, id string, uuid string, raw []byte, mdmContext *workflow.MDMContext) error
}

func SendCommandEvent(
	ctx context.Context,
	eventer MDMCommandEventer,
	filePath string,
	id string,
	commandUUID string,
) error {
	// read the response file
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading raw command: %w", err)
	}

	// make sure we can parse it
	genResp := new(mdmcommands.GenericResponse)
	err = plist.Unmarshal(raw, genResp)
	if err != nil {
		return fmt.Errorf("parsing generic command response: %w", err)
	}

	// check command UUID matches
	if want, have := commandUUID, genResp.CommandUUID; want != have {
		return fmt.Errorf("command UUIDs do not match: %s != %s", want, have)
	}

	// check UDID matches
	if want, have := &id, genResp.UDID; *want != *have {
		return fmt.Errorf("response UDIDs do not match: %s != %s", *want, *have)
	}

	// send the event
	return eventer.MDMCommandResponseEvent(ctx, id, commandUUID, raw, nil)
}
