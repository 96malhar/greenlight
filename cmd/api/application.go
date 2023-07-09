package main

import "log"

type config struct {
	port int
	env  string
}

type application struct {
	config config
	logger *log.Logger
}
