package foss

import (
	"context"
	"io"
	"time"

	"github.com/micromdm/nanocmd/workflow"
)

// MDMEventDumper is an MDM eventer middleware that dumps command responses to an output writer.
type MDMEventDumper struct {
	next   MDMEventReceiver
	output io.Writer
}

func NewMDMEventDumper(next MDMEventReceiver, output io.Writer) *MDMEventDumper {
	return &MDMEventDumper{next: next, output: output}
}

// MDMCommandResponseEvent dumps the raw command response and processes the next eventer.
func (d *MDMEventDumper) MDMCommandResponseEvent(ctx context.Context, id string, uuid string, raw []byte, mdmContext *workflow.MDMContext) error {
	d.output.Write(append(raw, '\n'))
	return d.next.MDMCommandResponseEvent(ctx, id, uuid, raw, mdmContext)
}

// MDMIdleEvent is called when an MDM Report Results has an "Idle" status.
func (d *MDMEventDumper) MDMIdleEvent(ctx context.Context, id string, raw []byte, mdmContext *workflow.MDMContext, eventAt time.Time) error {
	d.output.Write(append(raw, '\n'))
	return d.next.MDMIdleEvent(ctx, id, raw, mdmContext, eventAt)
}

// MDMCheckinEvent processes the next eventer.
func (d *MDMEventDumper) MDMCheckinEvent(ctx context.Context, id string, checkin interface{}, mdmContext *workflow.MDMContext) error {
	return d.next.MDMCheckinEvent(ctx, id, checkin, mdmContext)
}
