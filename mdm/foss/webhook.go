package foss

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/micromdm/nanocmd/logkeys"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

type Event struct {
	Topic     string    `json:"topic"`
	EventID   string    `json:"event_id"`
	CreatedAt time.Time `json:"created_at"`

	AcknowledgeEvent *AcknowledgeEvent `json:"acknowledge_event,omitempty"`
	CheckinEvent     *CheckinEvent     `json:"checkin_event,omitempty"`
}

type AcknowledgeEvent struct {
	UDID         string            `json:"udid,omitempty"`
	EnrollmentID string            `json:"enrollment_id,omitempty"`
	Status       string            `json:"status"`
	CommandUUID  string            `json:"command_uuid,omitempty"`
	Params       map[string]string `json:"url_params,omitempty"`
	RawPayload   []byte            `json:"raw_payload"`
}

type CheckinEvent struct {
	UDID         string            `json:"udid,omitempty"`
	EnrollmentID string            `json:"enrollment_id,omitempty"`
	Params       map[string]string `json:"url_params"`
	RawPayload   []byte            `json:"raw_payload"`

	// signals which tokenupdate this is to be able to tell whether this
	// is the initial enrollment vs. a following tokenupdate
	TokenUpdateTally *int `json:"token_update_tally,omitempty"`
}

// WebhookHandler parses the F/OSS MDM webhook callback for hand-off for futher processing.
func WebhookHandler(recv MDMEventReceiver, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)

		event := new(Event)
		if err := json.NewDecoder(r.Body).Decode(event); err != nil {
			logger.Info(logkeys.Message, "decoding body", logkeys.Error, err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		logger = logger.With(logsFromEvent(event)...)

		if event.Topic == "mdm.Connect" {
			if err := processAcknowledgeEvent(r.Context(), event.AcknowledgeEvent, recv); err != nil {
				logger.Info(logkeys.Message, "process acknowledge event", logkeys.Error, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		} else {
			if err := processCheckinEvent(r.Context(), event.Topic, event.CheckinEvent, recv); err != nil {
				logger.Info(logkeys.Message, "process checkin event", logkeys.Error, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}
		logger.Debug(logkeys.Message, "webhook event")
	}
}

func appendIfNotEmpty(slice *[]interface{}, key, value string) {
	if value != "" {
		*slice = append(*slice, key, value)
	}
}

func logsFromEvent(e *Event) (logs []interface{}) {
	if e == nil {
		return
	}
	logs = []interface{}{"topic", e.Topic}
	if e.AcknowledgeEvent != nil {
		appendIfNotEmpty(&logs, "udid", e.AcknowledgeEvent.UDID)
		appendIfNotEmpty(&logs, "enrollment_id", e.AcknowledgeEvent.EnrollmentID)
		appendIfNotEmpty(&logs, "status", e.AcknowledgeEvent.Status)
		appendIfNotEmpty(&logs, logkeys.CommandUUID, e.AcknowledgeEvent.CommandUUID)
	} else if e.CheckinEvent != nil {
		appendIfNotEmpty(&logs, "udid", e.CheckinEvent.UDID)
		appendIfNotEmpty(&logs, "enrollment_id", e.CheckinEvent.EnrollmentID)
	}

	return
}
