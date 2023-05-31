package foss

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/groob/plist"
	"github.com/micromdm/nanocmd/mdm"
	"github.com/micromdm/nanocmd/workflow"
)

type MDMCommandResponseEventer interface {
	MDMCommandResponseEvent(ctx context.Context, id string, uuid string, raw []byte, mdmContext *workflow.MDMContext) error
}

type MDMCheckinEventer interface {
	MDMCheckinEvent(ctx context.Context, id string, checkin interface{}, mdmContext *workflow.MDMContext) error
}

type MDMEventReceiver interface {
	MDMCommandResponseEventer
	MDMCheckinEventer
}

func idAndContext(udid, eid string, params map[string]string) (id string, mdmContext *workflow.MDMContext) {
	id = udid
	if id == "" {
		id = eid
	}
	if len(params) > 0 {
		mdmContext = &workflow.MDMContext{Params: params}
	}
	return
}

func processAcknowledgeEvent(ctx context.Context, e *AcknowledgeEvent, ev MDMCommandResponseEventer) error {
	if e == nil {
		return errors.New("empty acknowledge event")
	}
	if e.Status == "Idle" || e.CommandUUID == "" {
		return nil
	}
	id, mdmContext := idAndContext(e.UDID, e.EnrollmentID, e.Params)
	return ev.MDMCommandResponseEvent(ctx, id, e.CommandUUID, e.RawPayload, mdmContext)
}

func processCheckinEvent(ctx context.Context, topic string, e *CheckinEvent, ev MDMCheckinEventer) error {
	if e == nil {
		return errors.New("empty checkin event")
	}
	if !strings.HasPrefix(topic, "mdm.") {
		// we're assuming the topic is just a prefixed MessageType
		return errors.New("checkin topic incorrect prefix")
	}
	topic = topic[4:]
	checkin := mdm.NewCheckinFromMessageType(topic)
	if checkin == nil {
		return fmt.Errorf("no checkin type for message type: %s", topic)
	}
	if err := plist.Unmarshal(e.RawPayload, checkin); err != nil {
		return fmt.Errorf("unmarshal checkin: %w", err)
	}
	if tu, ok := checkin.(*mdm.TokenUpdate); ok && e.TokenUpdateTally != nil {
		// wrap the token update to include our enrolling status
		tue := &mdm.TokenUpdateEnrolling{TokenUpdate: tu}
		if *e.TokenUpdateTally == 1 {
			tue.Enrolling = true
		}
		if !tue.Valid() {
			return fmt.Errorf("invalid token update wrapper")
		}
		// use the wrapped version
		checkin = tue
	}
	id, mdmContext := idAndContext(e.UDID, e.EnrollmentID, e.Params)
	return ev.MDMCheckinEvent(ctx, id, checkin, mdmContext)
}
