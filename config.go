package main

import (
	"github.com/kelseyhightower/envconfig"
	"log"
)

type ConfigurationSpec struct {
	Port              int    `default:"7312"`
	UiPublicUrl       string `envconfig:"UI_PUBLIC_URL"`
	FacebookAppId     string `split_words:"true"`
	FacebookAppSecret string `split_words:"true"`
	DatabaseUrl       string `default:"root:root@tcp(localhost:8889)/fiiskins" split_words:"true"`
}

var config ConfigurationSpec

func initConfig() {
	envconfig.MustProcess("login", &config)
	log.Printf("%+v\n", config)
}
