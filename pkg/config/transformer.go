package config

import "io"

// Transformer reads in yaml from a io.Reader and allows transformations
// to be executed on the yaml. It then allows for the yaml to be output to an
// io.Writer
type Transformer interface {
	// Read reads the config source from the io.Reader
	Read(source io.Reader) error
	// Write outputs the config to the io.Writer
	Write(dest io.Writer) error
	// Get returns the value from the config
	Get(path string) (interface{}, error)
	// GetSlice returns the slice that is at the specified path in the config
	GetSlice(path string) ([]string, error)
	// GetMap returns the map[string]interface{} that is at the specified path in the config
	GetMap(path string) (map[string]interface{}, error)
	// Transform takes a path like output.buffer.size and sets the value to overwrite
	// the value or add the value if it is does not exist at the specified target path
	Transform(targetPath string, value interface{}) error
}
