package foss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/micromdm/nanocmd/log"
	"github.com/micromdm/nanocmd/mdm"
	"github.com/micromdm/nanocmd/workflow"
)

type event struct {
	resp  bool
	id    string
	uuid  string
	raw   []byte
	chkin interface{}
	ctx   *workflow.MDMContext
}

type eventRecorder struct {
	events []event
}

func (r *eventRecorder) MDMCommandResponseEvent(ctx context.Context, id string, uuid string, raw []byte, mdmContext *workflow.MDMContext) error {
	r.events = append(r.events, event{
		resp: true,
		id:   id,
		uuid: uuid,
		raw:  raw,
		ctx:  mdmContext,
	})
	return nil
}

func (r *eventRecorder) MDMCheckinEvent(ctx context.Context, id string, checkin interface{}, mdmContext *workflow.MDMContext) error {
	r.events = append(r.events, event{
		resp:  false,
		id:    id,
		chkin: checkin,
		ctx:   mdmContext,
	})
	return nil
}

func TestWebhook(t *testing.T) {
	eventRec := &eventRecorder{}
	hf := WebhookHandler(eventRec, log.NopLogger)

	r, err := http.NewRequestWithContext(context.Background(), "GET", "/webhook", nil)
	if err != nil {
		t.Fatal(err)
	}

	r.Body, err = os.Open("testdata/tokenupdate.json")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()

	recorder := httptest.NewRecorder()

	hf.ServeHTTP(recorder, r)

	if have, want := len(eventRec.events), 1; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

	tEvent := eventRec.events[0]

	if have, want := tEvent.resp, false; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

	udid := "FF269FDC-7A93-5F12-A4B7-09923F0D1F7F"

	if have, want := tEvent.id, udid; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

	if have, want := tEvent.resp, false; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

	if tEvent.ctx != nil {
		t.Error("expected nil")
	}

	tu, ok := tEvent.chkin.(*mdm.TokenUpdateEnrolling)
	if ok && tu != nil {
		if have, want := tu.Enrolling, true; ok && have != want {
			t.Errorf("have: %v, want: %v", have, want)
		}
		if have, want := tu.UDID, udid; ok && have != want {
			t.Errorf("have: %v, want: %v", have, want)
		}
	} else {
		t.Error("incorrect type from parsed webhook")
	}
}
