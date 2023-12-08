package vertex

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type Step struct {
	Settings *settings.StepSettings
}

func (csf *Step) SetStreaming(b bool) {
	csf.Settings.Chat.Stream = b
}

func IsVertexEngine(engine string) bool {
	return strings.HasPrefix(engine, "vertex")
}

// unpackPbJSON parses the protobuf response payload, returning the given JSON key
// TOOD: just return the whole JSON doc
func unpackPbJSON(X *structpb.Value, key string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(protojson.Format(X)), &data); err != nil {
		fmt.Println(err)
		os.Exit(3)
	}

	contents, ok := data[key].(string)
	if !ok {
		fmt.Printf("Could not find key '%s' in JSON\n", key)
		os.Exit(7)
	}
	return contents
}

func (csf *Step) Start(
	ctx context.Context,
	messages []*geppetto_context.Message,
) (*steps.StepResult[string], error) {
	clientSettings := csf.Settings.Client
	if clientSettings == nil {
		return nil, steps.ErrMissingClientSettings
	}

	vertexSettings := csf.Settings.Vertex
	if vertexSettings == nil {
		return nil, errors.New("no vertex settings")
	}

	gcpProject := os.Getenv("GCP_PROJECT")
	if gcpProject == "" {
		fmt.Println("Please set a valid GCP_PROJECT in environment")
		os.Exit(1)
	}
	client, err := aiplatform.NewPredictionClient(ctx, option.WithEndpoint(*csf.Settings.Vertex.BaseURL))
	if err != nil {
		return nil, err
	}
	location := "us-central1"
	engine := ""

	chatSettings := csf.Settings.Chat
	if chatSettings.Engine != nil {
		engine = *chatSettings.Engine
	} else {
		return nil, errors.New("no engine specified")
	}

	temperature := 0.0
	if chatSettings.Temperature != nil {
		temperature = *chatSettings.Temperature
	}

	// TODO: only some Vertex models support topP
	// topP := 0.0
	// if chatSettings.TopP != nil {
	// 	topP = *chatSettings.TopP
	// }

	maxTokens := 32
	if chatSettings.MaxResponseTokens != nil {
		maxTokens = *chatSettings.MaxResponseTokens
	}

	n := 1
	if vertexSettings.N != nil {
		n = *vertexSettings.N
	}
	// stop := chatSettings.Stop

	instance, err := structpb.NewValue(map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	// TODO: handle multiple
	var prompt string
	// prompt := []string{}
	for _, msg := range messages {
		// prompt = append(prompt, msg.Text)
		prompt = msg.Text
	}

	model, found := strings.CutPrefix(engine, "vertex-")
	if !found {
		return nil, errors.New(fmt.Sprintf("Failed to parse model from engine %v", engine))
	}
	modelAndSuffix := strings.Split(model, "@")
	if len(modelAndSuffix) == 1 {
		// ugly hack, to ensure we have a suffix for model variant (001, 002)
		model = fmt.Sprintf("%v@%v", model, "002")
	}

	switch modelAndSuffix[0] {
	case "text-bison":
		instance, _ = structpb.NewValue(map[string]interface{}{
			"prompt": prompt,
		})
	case "code-bison":
		instance, _ = structpb.NewValue(map[string]interface{}{
			"prefix": prompt,
		})
	case "codechat-bison":
		message, _ := structpb.NewValue(map[string]interface{}{
			"content": prompt,
			"author":  "User",
		})
		// TODO: keep history of conversation
		messages := []*structpb.Value{
			message,
		}

		promptContext := viper.GetString("context")
		instance, _ = structpb.NewValue(map[string]interface{}{
			"context":  promptContext,
			"messages": messages,
		})

	default:
		fmt.Println("Valid models include: text-bison, code-bison, codechat-bison")
		os.Exit(11)
	}

	parameters, err := structpb.NewValue(map[string]interface{}{
		"temperature":     temperature,
		"maxOutputTokens": maxTokens,
		"candidateCount":  n,
	})
	if err != nil {
		return nil, err
	}

	request := &aiplatformpb.PredictRequest{
		Endpoint: fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", gcpProject, location, model),
		Instances: []*structpb.Value{
			instance,
		},
		Parameters: parameters,
	}

	resp, err := client.Predict(ctx, request)
	if err != nil {
		return nil, err
	}

	// TODO: Handle multiple responses
	var contents string
	for _, prediction := range resp.Predictions {
		contents = unpackPbJSON(prediction, "content")
		// fmt.Println(contents)
	}
	return steps.Resolve(string(contents)), nil

}

// Close is only called after the returned monad has been entirely consumed
func (csf *Step) Close(ctx context.Context) error {
	return nil
}
