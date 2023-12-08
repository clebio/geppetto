package vertex

// Vertex API does not use an API key. Instead, we rely on standard GCP auth in environment

import (
	_ "embed"

	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/pkg/errors"
)

type Settings struct {
	// How many choice to create for each prompt
	N       *int    `yaml:"n" glazed.parameter:"vertex-n"`
	BaseURL *string `yaml:"base_url,omitempty" glazed.parameter:"vertex-base-url"`
}

func NewSettings() *Settings {
	return &Settings{
		N:       nil,
		BaseURL: nil,
	}
}

func NewSettingsFromParsedLayer(layer *layers.ParsedParameterLayer) (*Settings, error) {
	if layer == nil {
		return nil, errors.New("layer is nil")
	}
	ret := NewSettings()
	err := ret.UpdateFromParsedLayer(layer)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (s *Settings) Clone() *Settings {
	return &Settings{
		N: s.N,
	}
}

func (s *Settings) UpdateFromParsedLayer(layer *layers.ParsedParameterLayer) error {
	if layer == nil {
		return errors.New("layer is nil")
	}
	_, ok := layer.Layer.(*ParameterLayer)
	if !ok {
		return layers.ErrInvalidParameterLayer{
			Name:     layer.Layer.GetName(),
			Expected: "vertex",
		}
	}

	err := parameters.InitializeStructFromParameters(s, layer.Parameters)
	if err != nil {
		return err
	}

	return nil
}

//go:embed "vertex.yaml"
var settingsYAML []byte

type ParameterLayer struct {
	*layers.ParameterLayerImpl `yaml:",inline"`
}

func NewParameterLayer(options ...layers.ParameterLayerOptions) (*ParameterLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(settingsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ParameterLayer{
		ParameterLayerImpl: ret,
	}, nil
}
