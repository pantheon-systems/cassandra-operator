package config

import (
	"fmt"
	"io"

	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

const (
	nonInitTransformerError = "cannot transform uninitialized transformer"
)

// YAMLTransformer reads yamls, transforms them, and writes out yamls
type YAMLTransformer struct {
	config      *viper.Viper
	initialized bool
}

// NewYAMLTransformer transforms yamls
func NewYAMLTransformer() Transformer {
	cfg := viper.New()
	cfg.SetConfigType("yaml")
	return &YAMLTransformer{
		config:      cfg,
		initialized: false,
	}
}

// Read reads the yaml source from the io.Reader
func (t *YAMLTransformer) Read(source io.Reader) error {
	err := t.config.ReadConfig(source)
	if err != nil {
		return err
	}
	t.initialized = true
	return nil
}

// Write outputs the yaml to the io.Writer
func (t *YAMLTransformer) Write(dest io.Writer) error {
	if !t.initialized {
		return fmt.Errorf(nonInitTransformerError)
	}

	c := t.config.AllSettings()
	bs, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("unable to marshal config to YAML: %v", err)
	}
	_, err = dest.Write(bs)
	return err
}

// Get returns the value from the yaml config
func (t *YAMLTransformer) Get(path string) (interface{}, error) {
	if !t.initialized {
		return nil, fmt.Errorf(nonInitTransformerError)
	}

	if !t.config.IsSet(path) {
		return nil, fmt.Errorf("key not found")
	}

	return t.config.Get(path), nil
}

// GetSlice returns the slice that is at the specified path, if no slice exists
// then the empty set is returned
func (t *YAMLTransformer) GetSlice(path string) ([]string, error) {
	if !t.initialized {
		return nil, fmt.Errorf(nonInitTransformerError)
	}

	if !t.config.IsSet(path) {
		return nil, fmt.Errorf("key not found")
	}

	return t.config.GetStringSlice(path), nil
}

// GetMap returns the map[string]interface{} that is at the specified path in the config
func (t *YAMLTransformer) GetMap(path string) (map[string]interface{}, error) {
	if !t.initialized {
		return nil, fmt.Errorf(nonInitTransformerError)
	}

	if !t.config.IsSet(path) {
		return nil, fmt.Errorf("key not found")
	}

	return t.config.GetStringMap(path), nil
}

// Transform takes a path like output.buffer.size and sets the value to overwrite
// the value or add the value if it is does not exist at the specified target path
func (t *YAMLTransformer) Transform(targetPath string, value interface{}) error {
	if !t.initialized {
		return fmt.Errorf(nonInitTransformerError)
	}

	t.config.Set(targetPath, value)
	return nil
}
