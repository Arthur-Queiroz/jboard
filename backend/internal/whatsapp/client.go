package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Sender abstrai o envio de uma mensagem de texto. O scheduler depende desta
// interface, não do EvolutionClient, permitindo um mock nos testes.
type Sender interface {
	Send(ctx context.Context, recipient, message string) error
}

// EvolutionClient é o client da Evolution API (Baileys, self-hosted).
//
// Endpoint usado: POST {baseURL}/message/sendText/{instance} com corpo
// {"number": "...", "text": "..."} e header apikey. A estrutura exata pode
// variar conforme a versão da Evolution API — validar contra a instância real.
type EvolutionClient struct {
	baseURL  string
	instance string
	apiKey   string
	http     *http.Client
}

func NewEvolutionClient(baseURL, instance, apiKey string) *EvolutionClient {
	return &EvolutionClient{
		baseURL:  baseURL,
		instance: instance,
		apiKey:   apiKey,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *EvolutionClient) Send(ctx context.Context, recipient, message string) error {
	body, err := json.Marshal(map[string]string{
		"number": recipient,
		"text":   message,
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/message/sendText/%s", c.baseURL, c.instance)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("apikey", c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("evolution api: status %d para %s", resp.StatusCode, recipient)
	}
	return nil
}
