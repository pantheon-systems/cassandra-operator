package config_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pantheon-systems/cassandra-operator/pkg/config"
	"github.com/stretchr/testify/assert"
)

const (
	input = `
Hacker: true
name: steve
hobbies:
- skateboarding
- snowboarding
- go
clothing:
    jacket: leather
    trousers: denim
age: 35
eyes : brown
beard: true`
)

func TestTransform_NotInitialized(t *testing.T) {
	obj := config.NewYAMLTransformer()

	err := obj.Transform("some.key.path", "some-value")
	assert.Error(t, err)
	assert.Equal(t, "cannot transform uninitialized transformer", err.Error())
}

func TestWrite_NotInitialized(t *testing.T) {
	obj := config.NewYAMLTransformer()

	err := obj.Write(bytes.NewBufferString(""))
	assert.Error(t, err)
	assert.Equal(t, "cannot transform uninitialized transformer", err.Error())
}

func TestGet_NotInitialized(t *testing.T) {
	obj := config.NewYAMLTransformer()

	value, err := obj.Get("some.key.path")
	assert.Error(t, err)
	assert.Nil(t, value)
	assert.Equal(t, "cannot transform uninitialized transformer", err.Error())
}

func TestGetSlice_NotInitialized(t *testing.T) {
	obj := config.NewYAMLTransformer()

	value, err := obj.GetSlice("some.key.path")
	assert.Error(t, err)
	assert.Nil(t, value)
	assert.Equal(t, "cannot transform uninitialized transformer", err.Error())
}

func TestGetMap_NotInitialized(t *testing.T) {
	obj := config.NewYAMLTransformer()

	value, err := obj.GetMap("some.key.path")
	assert.Error(t, err)
	assert.Nil(t, value)
	assert.Equal(t, "cannot transform uninitialized transformer", err.Error())
}

func TestRead_NotYAML(t *testing.T) {
	nonYAMLInput := "something that is not yaml"

	obj := config.NewYAMLTransformer()
	err := obj.Read(strings.NewReader(nonYAMLInput))
	assert.Error(t, err)
}

func TestReadWrite_ValidYAML(t *testing.T) {
	obj := config.NewYAMLTransformer()
	err := obj.Read(strings.NewReader(input))
	// validate a few keys
	assert.NoError(t, err)

	buffer := bytes.NewBufferString("")
	obj.Write(buffer)

	actual := buffer.String()
	assert.Contains(t, actual, "hacker: true")
	assert.Contains(t, actual, "clothing:\n  jacket: leather\n  trousers: denim")
	assert.Contains(t, actual, "age: 35")
}

func TestGet_SuccessAndNotFound(t *testing.T) {

	obj := config.NewYAMLTransformer()
	err := obj.Read(strings.NewReader(input))
	assert.NoError(t, err)

	value, err := obj.Get("name")
	assert.NoError(t, err)
	assert.Equal(t, "steve", value.(string))

	value, err = obj.Get("clothing.jacket")
	assert.NoError(t, err)
	assert.Equal(t, "leather", value.(string))

	_, err = obj.Get("clothing.pants")
	assert.Error(t, err)

	value, err = obj.Get("age")
	assert.NoError(t, err)
	assert.Equal(t, 35, value.(int))
}

func TestGetSlice_SuccessAndNotFound(t *testing.T) {
	obj := config.NewYAMLTransformer()
	err := obj.Read(strings.NewReader(input))
	assert.NoError(t, err)

	sliceValue, err := obj.GetSlice("hobbies")
	assert.NoError(t, err)
	assert.Equal(t, "skateboarding", sliceValue[0])

	_, err = obj.GetSlice("features")
	assert.Error(t, err)

	sliceValue, err = obj.GetSlice("eyes")
	assert.NoError(t, err)
	assert.Equal(t, []string([]string{"brown"}), sliceValue)
}

func TestGetMap_SuccessAndNotFound(t *testing.T) {
	obj := config.NewYAMLTransformer()
	err := obj.Read(strings.NewReader(input))
	assert.NoError(t, err)

	mapValue, err := obj.GetMap("clothing")
	assert.NoError(t, err)
	assert.Equal(t, "leather", mapValue["jacket"])

	_, err = obj.GetMap("features")
	assert.Error(t, err)

	mapValue, err = obj.GetMap("eyes")
	assert.NoError(t, err)
	assert.Empty(t, mapValue)
}

func TestTransform_Success(t *testing.T) {
	obj := config.NewYAMLTransformer()
	err := obj.Read(strings.NewReader(input))
	assert.NoError(t, err)

	err = obj.Transform("hobbies", []string{"newhobbie1", "newhobbie2"})
	assert.NoError(t, err)

	actualSlice, err := obj.GetSlice("hobbies")
	assert.NoError(t, err)
	assert.Len(t, actualSlice, 2)
	assert.Equal(t, "newhobbie1", actualSlice[0])

	err = obj.Transform("name", "bob")
	assert.NoError(t, err)

	actual, err := obj.Get("name")
	assert.NoError(t, err)
	assert.Equal(t, "bob", actual)

	err = obj.Transform("age", 25)
	assert.NoError(t, err)

	actualInt, err := obj.Get("age")
	assert.NoError(t, err)
	assert.Equal(t, 25, actualInt)
}
