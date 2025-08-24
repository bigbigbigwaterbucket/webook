package config

import (
	"fmt"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("dev")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./webook/internal/config")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	fmt.Println(viper.GetString("k8s.db.mysql.dsn"))
}
