package api

// Anotações gerais da especificação OpenAPI (lidas pelo swag).
//
//	@title			Students API
//	@version		1.0
//	@description	API RESTful para gerenciar estudantes (CRUD), com validação de
//	@description	CPF (dígitos verificadores + unicidade) e e-mail, listagem
//	@description	paginada, cache e observabilidade.
//	@description
//	@description	Erros são retornados como JSON no formato `{"error": "mensagem"}`
//	@description	com o status HTTP adequado (400 inválido, 404 não encontrado,
//	@description	409 CPF duplicado).
//	@contact.name	Students API
//	@license.name	MIT
//	@BasePath		/
//	@schemes		http https

import (
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"

	// Registra a especificação gerada por `swag init` (efeito colateral do init()).
	_ "github.com/apolinario0x21/students-api/docs"
)

// registerSwagger publica a UI interativa do Swagger em /swagger/index.html.
func registerSwagger(e *echo.Echo) {
	e.GET("/swagger/*", echoSwagger.WrapHandler)
}
