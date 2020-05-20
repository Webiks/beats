package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type ConfigYaml struct {
	Query  []InnerConfig     `yaml:"hardware_query"`
	Format InnerConfigFormat `yaml:"output_format"`
}

type InnerConfig struct {
	TypeOf string `yaml:"type"`
	Name   string `yaml:"name"`
}

type InnerConfigFormat struct {
	UseType  bool `yaml:"use_type_as_key"`
	UseConst bool `yaml:"use_constant_key"`
}

func ReadFile(cfg *ConfigYaml) {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	fmt.Println(exPath, "exec path")

	filename, _ := filepath.Abs(exPath + `\hardware.yml`)
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		fmt.Println(err)
	}

	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		fmt.Println(err)
	}

}

func B2s(bs []int8) string {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}
	return string(b)
}
