package main

//go:generate swag init -g swagger.go -o docs --parseDependency --parseInternal

// @title Switchboard Backend API
// @version 0.1.0
// @description Multi-host Docker monitoring API for container status and proxy mappings.
// @host localhost:8069
// @BasePath /
// @schemes http
