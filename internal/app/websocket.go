package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/formation-res/open-location-hub-cli/internal/cli"
	"github.com/formation-res/open-location-hub-cli/internal/output"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

type wsEnvelope struct {
	Event          string         `json:"event"`
	Topic          string         `json:"topic,omitempty"`
	SubscriptionID *int           `json:"subscription_id,omitempty"`
	Payload        any            `json:"payload,omitempty"`
	Params         map[string]any `json:"params,omitempty"`
	Code           *int           `json:"code,omitempty"`
	Description    string         `json:"description,omitempty"`
	ReceivedAt     string         `json:"received_at,omitempty"`
}

type wsNDJSONRecord struct {
	ReceivedAt string     `json:"received_at"`
	Topic      string     `json:"topic"`
	Message    wsEnvelope `json:"message"`
}

func websocketCommand(cfg *cli.Config, printer *output.Printer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ws",
		Short: "WebSocket subscribe and publish commands",
		Long:  "Connects to GET /v2/ws/socket for topic subscriptions and inbound publish operations such as location_updates.",
	}
	cmd.AddCommand(wsSubscribeCommand(cfg, printer))
	cmd.AddCommand(wsPublishCommand(cfg, printer))
	return cmd
}

func wsSubscribeCommand(cfg *cli.Config, printer *output.Printer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribe --topic location_updates",
		Short: "Subscribe to a websocket topic",
		RunE: func(cmd *cobra.Command, args []string) error {
			topic, _ := cmd.Flags().GetString("topic")
			paramsFlag, _ := cmd.Flags().GetStringArray("param")
			wsURL, _ := cmd.Flags().GetString("ws-url")
			if err := cfg.EnsureToken(cmd.Context()); err != nil {
				return err
			}

			return runWebsocketSubscription(cmd.Context(), cfg, topic, wsURL, paramsFlag, func(envelope wsEnvelope) error {
				if printer.JSON {
					return printer.PrintLine(envelope)
				}
				return printer.Print(envelope)
			}, func() {
				printer.Info("subscribed to %s", topic)
			})
		},
	}
	cmd.Flags().String("topic", "", "Topic name, e.g. location_updates, fence_events, metadata_changes")
	cmd.Flags().StringArray("param", nil, "Topic filter in key=value form. Repeat as needed")
	cmd.Flags().String("ws-url", "", "Explicit websocket URL. Defaults to <base-url>/v2/ws/socket")
	must(cmd.MarkFlagRequired("topic"))
	return cmd
}

func wsNDJSONStreamCommand(cfg *cli.Config, printer *output.Printer, use, short, topic string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  fmt.Sprintf("Subscribe to %s and emit NDJSON records to stdout.", topic),
		RunE: func(cmd *cobra.Command, args []string) error {
			paramsFlag, _ := cmd.Flags().GetStringArray("param")
			wsURL, _ := cmd.Flags().GetString("ws-url")
			encoder := json.NewEncoder(printer.Out)
			encoder.SetEscapeHTML(false)
			return runWebsocketSubscription(cmd.Context(), cfg, topic, wsURL, paramsFlag, func(envelope wsEnvelope) error {
				if envelope.Event == "subscribed" {
					return nil
				}
				record := wsNDJSONRecord{
					ReceivedAt: time.Now().UTC().Format(time.RFC3339Nano),
					Topic:      topic,
					Message:    envelope,
				}
				return encoder.Encode(record)
			}, func() {
				printer.Info("streaming %s as NDJSON", topic)
			})
		},
	}
	cmd.Flags().StringArray("param", nil, "Topic filter in key=value form. Repeat as needed")
	cmd.Flags().String("ws-url", "", "Explicit websocket URL. Defaults to <base-url>/v2/ws/socket")
	return cmd
}

func wsPublishCommand(cfg *cli.Config, printer *output.Printer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish --topic location_updates --file payload.json",
		Short: "Publish a websocket message to a topic",
		RunE: func(cmd *cobra.Command, args []string) error {
			topic, _ := cmd.Flags().GetString("topic")
			file, _ := cmd.Flags().GetString("file")
			paramsFlag, _ := cmd.Flags().GetStringArray("param")
			wsURL, _ := cmd.Flags().GetString("ws-url")
			if err := cfg.EnsureToken(cmd.Context()); err != nil {
				return err
			}

			payloadBytes, err := readPayload(file)
			if err != nil {
				return err
			}
			var payload any
			if err := json.Unmarshal(payloadBytes, &payload); err != nil {
				return err
			}
			params, err := parseParams(paramsFlag)
			if err != nil {
				return err
			}
			if cfg.Token != "" {
				params["token"] = cfg.Token
			}

			conn, err := dialWebsocket(cmd.Context(), cfg, wsURL)
			if err != nil {
				return err
			}
			defer conn.Close()

			msg := wsEnvelope{
				Event:   "message",
				Topic:   topic,
				Payload: payload,
				Params:  params,
			}
			if err := conn.WriteJSON(msg); err != nil {
				return err
			}
			if printer.JSON {
				return printer.Print(map[string]any{"published": true, "topic": topic})
			}
			printer.Success("published websocket message to %s", topic)
			return nil
		},
	}
	cmd.Flags().String("topic", "", "Topic name")
	cmd.Flags().StringP("file", "f", "", "Read payload from JSON file or - for stdin")
	cmd.Flags().StringArray("param", nil, "Message params in key=value form. Repeat as needed")
	cmd.Flags().String("ws-url", "", "Explicit websocket URL. Defaults to <base-url>/v2/ws/socket")
	must(cmd.MarkFlagRequired("topic"))
	must(cmd.MarkFlagRequired("file"))
	return cmd
}

func parseParams(items []string) (map[string]any, error) {
	params := map[string]any{}
	for _, item := range items {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --param %q, want key=value", item)
		}
		params[parts[0]] = parts[1]
	}
	return params, nil
}

func runWebsocketSubscription(parentCtx context.Context, cfg *cli.Config, topic, wsURL string, paramsFlag []string, handle func(wsEnvelope) error, onSubscribe func()) error {
	if err := cfg.EnsureToken(parentCtx); err != nil {
		return err
	}

	params, err := parseParams(paramsFlag)
	if err != nil {
		return err
	}
	if cfg.Token != "" {
		params["token"] = cfg.Token
	}

	ctx, stop := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, err := dialWebsocket(ctx, cfg, wsURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	subscribe := wsEnvelope{Event: "subscribe", Topic: topic, Params: params}
	if err := conn.WriteJSON(subscribe); err != nil {
		return err
	}
	if onSubscribe != nil {
		onSubscribe()
	}

	for {
		envelope, err := readWSEnvelope(conn)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		if err := handle(envelope); err != nil {
			return err
		}
	}
}

func readWSEnvelope(conn *websocket.Conn) (wsEnvelope, error) {
	var envelope wsEnvelope
	if err := conn.ReadJSON(&envelope); err != nil {
		return wsEnvelope{}, err
	}
	envelope.ReceivedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return envelope, nil
}

func dialWebsocket(ctx context.Context, cfg *cli.Config, explicit string) (*websocket.Conn, error) {
	wsURL, err := deriveWebsocketURL(cfg.BaseURL, explicit)
	if err != nil {
		return nil, err
	}
	dialer := websocket.Dialer{
		HandshakeTimeout: cfg.Timeout,
	}
	header := http.Header{}
	conn, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func deriveWebsocketURL(baseURL, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported base URL scheme %q", u.Scheme)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/v2/ws/socket"
	return u.String(), nil
}
