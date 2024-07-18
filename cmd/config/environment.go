package config

import "github.com/vrischmann/envconfig"

var Env struct {
	MongoDB struct {
		URL      string `envconfig:"default=mongodb+srv://sergingroisman:0xlBYnLOkri80XvS@meal-maker-db-cluster-1.rabnin5.mongodb.net/?retryWrites=true&w=majority&appName=meal-maker-db-cluster-1"`
		Database string `envconfig:"default=meal-maker-db"`
	}

	Auth struct {
		SecretKey string `envconfig:"default=your-secret-key"`
	}
}

func Init() error {
	return envconfig.Init(&Env)
}
