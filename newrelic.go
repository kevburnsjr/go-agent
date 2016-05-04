package newrelic

import (
	"go.datanerd.us/p/will/go-sdk/api"
	"go.datanerd.us/p/will/go-sdk/internal"
)

func NewConfig(appname, license string) api.Config {
	return api.NewConfig(appname, license)
}

type Application api.Application
type Transaction api.Transaction

func NewApplication(c api.Config) (Application, error) {
	return internal.NewApp(c)
}
