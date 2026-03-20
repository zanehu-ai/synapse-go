package ratelimit

import "encoding/json"

// EstimateTotalTokens provides a rough estimate of total tokens from the request body.
// Uses the approximation: ~4 characters per token for input, max_tokens or 500 for output.
func EstimateTotalTokens(body []byte) int {
	return estimateInputTokens(body) + estimateOutputTokens(body)
}

const maxInputTokenEstimate = 128000

func estimateInputTokens(body []byte) int {
	var req struct {
		Messages []struct {
			Content interface{} `json:"content"`
		} `json:"messages"`
	}

	totalChars := 0
	if err := json.Unmarshal(body, &req); err == nil {
		for _, m := range req.Messages {
			switch c := m.Content.(type) {
			case string:
				totalChars += len(c)
			default:
				b, _ := json.Marshal(c)
				totalChars += len(b)
			}
		}
	}

	if totalChars == 0 {
		totalChars = len(body)
	}

	tokens := totalChars / 4
	if tokens < 1 {
		tokens = 1
	}
	if tokens > maxInputTokenEstimate {
		tokens = maxInputTokenEstimate
	}
	return tokens
}

func estimateOutputTokens(body []byte) int {
	var req struct {
		MaxTokens *int `json:"max_tokens"`
	}

	if err := json.Unmarshal(body, &req); err == nil && req.MaxTokens != nil && *req.MaxTokens > 0 {
		if *req.MaxTokens > 500 {
			return 500 // cap estimate for rate limiting
		}
		return *req.MaxTokens
	}

	return 500
}
