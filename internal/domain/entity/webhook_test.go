package entity

import (
	"testing"
)

func TestWebhookRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     WebhookRequest
		wantErr error
	}{
		{
			name: "valid request",
			req: WebhookRequest{
				User:   "user1",
				Asset:  "BTC",
				Amount: "100.5",
			},
			wantErr: nil,
		},
		{
			name: "missing user",
			req: WebhookRequest{
				User:   "",
				Asset:  "BTC",
				Amount: "100.5",
			},
			wantErr: ErrMissingUser,
		},
		{
			name: "missing asset",
			req: WebhookRequest{
				User:   "user1",
				Asset:  "",
				Amount: "100.5",
			},
			wantErr: ErrMissingAsset,
		},
		{
			name: "missing amount",
			req: WebhookRequest{
				User:   "user1",
				Asset:  "BTC",
				Amount: "",
			},
			wantErr: ErrMissingAmount,
		},
		{
			name: "all fields missing",
			req: WebhookRequest{
				User:   "",
				Asset:  "",
				Amount: "",
			},
			wantErr: ErrMissingUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if err != tt.wantErr {
				t.Errorf("WebhookRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
