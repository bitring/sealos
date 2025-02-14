package yaml_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/labring/sealos/pkg/utils/yaml"
)

type testStruct struct {
	Name  string `yaml:"name"`
	Value int    `yaml:"value"`
}

func TestUnmarshalStrict(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		obj     interface{}
		wantErr bool
	}{
		{
			name:  "valid yaml",
			input: "name: test\nvalue: 123",
			obj:   &testStruct{},
		},
		{
			name:    "non-pointer",
			input:   "name: test",
			obj:     testStruct{},
			wantErr: true,
		},
		{
			name:    "non-struct pointer",
			input:   "test",
			obj:     new(string),
			wantErr: true,
		},
		{
			name:    "invalid yaml",
			input:   "invalid: [",
			obj:     &testStruct{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader([]byte(tt.input))
			err := yaml.Unmarshal(r, tt.obj)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnmarshalToMap(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name:  "valid yaml",
			input: []byte("key: value\nnum: 123"),
			want: map[string]interface{}{
				"key": "value",
				"num": float64(123), // YAML numbers are unmarshaled as float64
			},
		},
		{
			name:    "invalid yaml",
			input:   []byte("invalid: ["),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.UnmarshalToMap(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestToJSON(t *testing.T) {
	input := []byte(`
name: test1
value: 1
---
name: test2
value: 2
`)
	jsons := yaml.ToJSON(input)
	assert.Len(t, jsons, 2)
	assert.Contains(t, jsons[0], `"name":"test1"`)
	assert.Contains(t, jsons[1], `"name":"test2"`)
}

func TestMarshalAndUnmarshal(t *testing.T) {
	obj := testStruct{
		Name:  "test",
		Value: 123,
	}

	data, err := yaml.Marshal(obj)
	require.NoError(t, err)

	var decoded testStruct
	err = yaml.Unmarshal(bytes.NewReader(data), &decoded)
	require.NoError(t, err)

	assert.Equal(t, obj, decoded)
}

func TestMarshalFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.yaml")

	obj := testStruct{
		Name:  "test",
		Value: 123,
	}

	err := yaml.MarshalFile(file, obj)
	require.NoError(t, err)

	content, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Name: test")
	assert.Contains(t, string(content), "Value: 123")
}

func TestIsNil(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    bool
		wantErr bool
	}{
		{
			name:  "empty map",
			input: []byte("{}"),
			want:  true,
		},
		{
			name:  "non-empty map",
			input: []byte("key: value"),
			want:  false,
		},
		{
			name:    "invalid yaml",
			input:   []byte("invalid: ["),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.IsNil(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUnmarshalFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.yaml")

	content := []byte("name: test\nvalue: 123")
	err := os.WriteFile(file, content, 0644)
	require.NoError(t, err)

	var obj testStruct
	err = yaml.UnmarshalFile(file, &obj)
	require.NoError(t, err)
	assert.Equal(t, "test", obj.Name)
	assert.Equal(t, 123, obj.Value)
}

func TestMarshalConfigs(t *testing.T) {
	obj1 := testStruct{Name: "test1", Value: 1}
	obj2 := testStruct{Name: "test2", Value: 2}

	data, err := yaml.MarshalConfigs(obj1, obj2)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "Name: test1")
	assert.Contains(t, content, "Name: test2")
	assert.Contains(t, content, "---")
}

func TestShowStructYaml(t *testing.T) {
	obj := testStruct{
		Name:  "test",
		Value: 123,
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	yaml.ShowStructYaml(obj)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Name: test")
	assert.Contains(t, output, "Value: 123")
}
