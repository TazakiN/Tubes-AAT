package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Server    ServerConfig    `json:"server"`
	Database  DatabaseConfig  `json:"database"`
	RabbitMQ  RabbitMQConfig  `json:"rabbitmq"`
	Anonymous AnonymousConfig `json:"anonymous"`
}

type ServerConfig struct {
	Port string `json:"port"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
}

type RabbitMQConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type AnonymousConfig struct {
	Salt string `json:"salt"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
