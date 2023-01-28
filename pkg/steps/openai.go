package steps

import (
	"context"
	"github.com/PullRequestInc/go-gpt3"
	"gopkg.in/errgo.v2/fmt/errors"
	"time"
)

type OpenAICompletionStepState int

const (
	OpenAICompletionStepNotStarted OpenAICompletionStepState = iota
	OpenAICompletionStepRunning
	OpenAICompletionStepFinished
	OpenAICompletionStepClosed
)

type OpenAICompletionStep struct {
	output chan Result[string]
	state  OpenAICompletionStepState
	apiKey string
}

func NewOpenAICompletionStep(apiKey string) *OpenAICompletionStep {
	return &OpenAICompletionStep{
		output: nil,
		apiKey: apiKey,
		state:  OpenAICompletionStepNotStarted,
	}
}

func (o *OpenAICompletionStep) Start(ctx context.Context, prompt string) error {
	o.output = make(chan Result[string])

	o.state = OpenAICompletionStepRunning
	go func() {
		defer func() {
			o.state = OpenAICompletionStepClosed
			close(o.output)
		}()

		client := gpt3.NewClient(o.apiKey, gpt3.WithTimeout(120*time.Second))

		temperature := float32(0.7)
		maxTokens := 2048
		topP := float32(1.0)
		n := 1
		logProbs := 0
		stream := false
		stop := []string{}
		presencePenalty := float32(0.0)
		frequencyPenalty := float32(0.0)

		completion, err := client.CompletionWithEngine(ctx, "text-davinci-003", gpt3.CompletionRequest{
			Prompt:           []string{prompt},
			MaxTokens:        &maxTokens,
			Temperature:      &temperature,
			TopP:             &topP,
			N:                &n,
			LogProbs:         &logProbs,
			Echo:             false,
			Stop:             stop,
			PresencePenalty:  presencePenalty,
			FrequencyPenalty: frequencyPenalty,
			Stream:           stream,
		})
		o.state = OpenAICompletionStepFinished

		if err != nil {
			o.output <- Result[string]{err: err}
			return
		}

		if len(completion.Choices) == 0 {
			o.output <- Result[string]{err: errors.Newf("no choices returned from OpenAI")}
			return
		}

		o.output <- Result[string]{value: completion.Choices[0].Text}
	}()

	return nil
}

func (o *OpenAICompletionStep) GetOutput() <-chan Result[string] {
	return o.output
}

func (o *OpenAICompletionStep) GetState() interface{} {
	return o.state
}

func (o *OpenAICompletionStep) IsFinished() bool {
	return o.state == OpenAICompletionStepFinished
}
