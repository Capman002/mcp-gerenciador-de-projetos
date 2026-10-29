package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// GetProposalText baixa a página de proposta já renderizada pelo portal e devolve
// todo o texto dela, sem marcação.
func GetProposalText(ctx context.Context, cfg *Config, page string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", cfg.ApiUrl+"/propostas/"+page, nil)
	if err != nil {
		return "", fmt.Errorf("request error: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("página /propostas/%s respondeu %d", page, resp.StatusCode)
	}
	return htmlToText(string(body)), nil
}

var (
	reHidden = regexp.MustCompile(`(?is)<(head|script|style|svg|noscript)\b.*?</(head|script|style|svg|noscript)>`)
	reItem   = regexp.MustCompile(`(?i)<li\b[^>]*>`)
	reCell   = regexp.MustCompile(`(?i)</t[dh]>`)
	reBlock  = regexp.MustCompile(`(?i)<(br|/?(p|div|ul|ol|h[1-6]|tr|table|section|article|header|footer|main|aside|nav|details|summary|blockquote|figure|figcaption|dl|dt|dd|label|button))\b[^>]*>`)
	reTag    = regexp.MustCompile(`<[^>]*>`)
	reSpaces = regexp.MustCompile(`[ \t\r\f\v\x{00a0}]+`)
	reBlank  = regexp.MustCompile(`\n( ?\n)+`)
)

// htmlToText remove a marcação mantendo todo o conteúdo visível: blocos viram
// linhas, itens de lista viram "- " e células de tabela são separadas por " | ".
func htmlToText(page string) string {
	page = reHidden.ReplaceAllString(page, "")
	page = reItem.ReplaceAllString(page, "\n- ")
	page = reCell.ReplaceAllString(page, " | ")
	page = reBlock.ReplaceAllString(page, "\n")
	page = html.UnescapeString(reTag.ReplaceAllString(page, ""))
	page = reSpaces.ReplaceAllString(page, " ")
	page = reBlank.ReplaceAllString(page, "\n\n")
	return strings.TrimSpace(page)
}

// CallAction sends a structured action to /api/mcp/action.
func CallAction(ctx context.Context, cfg *Config, entity, action string, data map[string]any) (string, error) {
	payload := map[string]any{
		"entity": entity,
		"action": action,
		"data":   data,
	}
	return doPost(ctx, cfg, "/api/mcp/action", payload)
}

// CallQuery sends a SQL query to /api/mcp/query.
func CallQuery(ctx context.Context, cfg *Config, query string) (string, error) {
	payload := map[string]any{
		"query": query,
	}
	return doPost(ctx, cfg, "/api/mcp/query", payload)
}

// doPost is the internal HTTP helper.
func doPost(ctx context.Context, cfg *Config, path string, payload map[string]any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cfg.ApiUrl+path, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read error: %w", err)
	}
	return string(respBody), nil
}
