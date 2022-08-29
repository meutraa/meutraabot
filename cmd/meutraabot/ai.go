package main

type CompletionResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Text string `json:"text"`
}

type CompletionRequest struct {
	Prompt           string         `json:"prompt"`
	MaxTokens        int            `json:"max_tokens"`
	Temperature      float64        `json:"temperature"`
	N                int            `json:"n"`
	TopP             float64        `json:"top_p"`
	PresencePenalty  float64        `json:"presence_penalty"`
	FrequencyPenalty float64        `json:"frequency_penalty"`
	LogitBias        map[string]int `json:"logit_bias"`
	Stop             string         `json:"stop"`
	User             string         `json:"user"`
}
