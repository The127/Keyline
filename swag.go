package docsgen

//go:generate go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go -o docs -d . --parseDepth 3 --parseDependency --parseInternal
