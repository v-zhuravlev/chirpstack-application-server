package mydevices

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	pb "github.com/brocaar/chirpstack-api/go/as/integration"
	"github.com/brocaar/chirpstack-application-server/internal/integration"
	"github.com/brocaar/chirpstack-application-server/internal/logging"
	"github.com/brocaar/lorawan"
)

// Config contains the configuration for the MyDevices endpoint.
type Config struct {
	URL string `json:"url"`
}

// Integration implements a MyDevices integration.
type Integration struct {
	config Config
}

type uplinkPayload struct {
	CorrelationID interface{}   `json:"correlationID"`
	DevEUI        lorawan.EUI64 `json:"devEUI"`
	Data          []byte        `json:"data"`
	FCnt          uint32        `json:"fCnt"`
	FPort         uint32        `json:"fPort"`
	RXInfo        []rxInfo      `json:"rxInfo"`
}

type rxInfo struct {
	GatewayID lorawan.EUI64 `json:"gatewayID"`
	RSSI      int32         `json:"rssi"`
	LoRaSNR   float64       `json:"loRaSNR"`
	Location  location      `json:"location"`
}

type location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// New creates a new MyDevices integration.
func New(conf Config) (*Integration, error) {
	return &Integration{
		config: conf,
	}, nil
}

func (i *Integration) send(url string, msg interface{}) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshal json error")
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return errors.Wrap(err, "new request error")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "http request error")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("expected 2xx response, got: %d", resp.StatusCode)
	}

	return nil
}

// Close closes the handler.
func (i *Integration) Close() error {
	return nil
}

// SendDataUp sends a data-up payload.
func (i *Integration) SendDataUp(ctx context.Context, vars map[string]string, pl pb.UplinkEvent) error {
	if pl.FPort == 0 {
		return nil
	}

	up := uplinkPayload{
		CorrelationID: ctx.Value(logging.ContextIDKey),
		Data:          pl.Data,
		FCnt:          pl.FCnt,
		FPort:         pl.FPort,
	}
	copy(up.DevEUI[:], pl.DevEui)

	for i := range pl.RxInfo {
		ri := rxInfo{
			RSSI:    pl.RxInfo[i].Rssi,
			LoRaSNR: pl.RxInfo[i].LoraSnr,
			Location: location{
				Latitude:  pl.RxInfo[i].GetLocation().Latitude,
				Longitude: pl.RxInfo[i].GetLocation().Longitude,
			},
		}
		copy(ri.GatewayID[:], pl.RxInfo[i].GatewayId)

		up.RXInfo = append(up.RXInfo, ri)
	}

	log.WithFields(log.Fields{
		"dev_eui": up.DevEUI,
		"ctx_id":  ctx.Value(logging.ContextIDKey),
		"url":     i.config.URL,
		"event":   "up",
	}).Info("integration/mydevices: publishing event")

	if err := i.send(i.config.URL, up); err != nil {
		return errors.Wrap(err, "send event error")
	}

	return nil
}

// SendJoinNotification sends a join notification.
func (i *Integration) SendJoinNotification(ctx context.Context, vars map[string]string, pl pb.JoinEvent) error {
	return nil
}

// SendACKNotification sends an ACK notification.
func (i *Integration) SendACKNotification(ctx context.Context, vars map[string]string, pl pb.AckEvent) error {
	return nil
}

// SendErrorNotification sends an error notification.
func (i *Integration) SendErrorNotification(ctx context.Context, vars map[string]string, pl pb.ErrorEvent) error {
	return nil
}

// SendStatusNotification sends a status notification.
func (i *Integration) SendStatusNotification(ctx context.Context, vars map[string]string, pl pb.StatusEvent) error {
	return nil
}

// SendLocationNotification sends a location notification.
func (i *Integration) SendLocationNotification(ctx context.Context, vars map[string]string, pl pb.LocationEvent) error {
	return nil
}

// DataDownChan returns nil.
func (i *Integration) DataDownChan() chan integration.DataDownPayload {
	return nil
}
