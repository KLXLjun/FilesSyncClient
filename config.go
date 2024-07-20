package main

import (
	"path"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ClientConf ClientConfig `yaml:"client"`
	ServerConf ServerConfig `yaml:"server"`
}

type ClientConfig struct {
	Root      string   `yaml:"root"`
	WhiteList []string `yaml:"whitelist"`
}

type ServerConfig struct {
	Url   string `yaml:"url"`
	Check string `yaml:"check"`
}

var conf = Config{
	ClientConf: ClientConfig{
		Root:      "./",
		WhiteList: []string{"example.jar"},
	},
	ServerConf: ServerConfig{
		Url:   "http://example.com",
		Check: "example",
	},
}

func LoadConf(run string) (bool, error) {
	bytes, _ := yaml.Marshal(conf)
	data, err := ReadOrCreateFile(path.Join(run, "config.yaml"), bytes)
	if err != nil {
		logrus.Warn("配置文件 config.yaml 不存在,现在已创建")
	}
	err2 := yaml.Unmarshal(data, &conf)
	if err2 != nil {
		return false, err2
	}
	return true, nil
}
