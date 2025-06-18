# Crewhu Observability Go

Biblioteca de observabilidade para serviços Go da CrewHU, fornecendo instrumentação padronizada com OpenTelemetry para tracing distribuído e logging estruturado.

## Visão Geral

Esta biblioteca simplifica a implementação do OpenTelemetry em aplicações Go, permitindo monitorar e rastrear o comportamento dos serviços em ambientes de produção. Inicialmente focada em medição de tempo de execução, agora oferece recursos completos de tracing e logging.

## Recursos

- **Tracing distribuído**: Rastreamento de requisições através de múltiplos serviços
- **Logging estruturado**: Logs padronizados e estruturados com contexto de tracing
- **Instrumentação HTTP**: Suporte para instrumentação automática de APIs HTTP (com middleware para Fiber)
- **Exportação via REST API**: Envio de telemetria via protocolo HTTP/OTLP

## Instalação

```bash
go get github.com/crewhu/observability_go
```

Para usar uma versão específica:

```bash
go get github.com/crewhu/observability_go@v1.0.0
```

## Configuração

### Variáveis de Ambiente Necessárias

Para utilizar a biblioteca, configure as seguintes variáveis de ambiente:

- `OTEL_PROJECT_NAME`: Nome do projeto/serviço (usado como Service.Name nos dados de telemetria)
- `OTEL_ENDPOINT`: Endpoint do coletor OpenTelemetry (ex: "http://otel-collector:4318")

### Configuração em Arquivos

#### Desenvolvimento (.env)

```
OTEL_PROJECT_NAME=meu-servico
OTEL_ENDPOINT=http://localhost:4318
```

#### Produção (oni.yaml)

```yaml
env:
  - name: OTEL_PROJECT_NAME
    value: meu-servico
  - name: OTEL_ENDPOINT
    value: http://otel-collector:4318
```

## Uso Básico

### Inicialização no main.go

```go
package main

import (
	"context"
	"os"
	
	logging "github.com/crewhu/observability_go/pkg/logging"
	tracing "github.com/crewhu/observability_go/pkg/tracing"
	"go.opentelemetry.io/otel"
)

func main() {
	ctx := context.Background()
	
	// Inicialização do logging
	logging.SetLoggingLevel(logging.LogLevelInfo)
	logging.InitLoggerCollector(os.Getenv("OTEL_PROJECT_NAME"), os.Getenv("OTEL_ENDPOINT"))
	
	// Inicialização do tracing
	traceProvider, err := tracing.NewTracer(os.Getenv("OTEL_PROJECT_NAME"), os.Getenv("OTEL_ENDPOINT"))
	if err != nil {
		logging.Error(ctx, "erro ao inicializar tracer: %v", err)
	}
	defer traceProvider.Shutdown(ctx)
	
	// Configura o tracer global
	otel.SetTracerProvider(traceProvider.GetProvider())
	
	// Resto da aplicação...
}
```

### Middleware para Fiber HTTP

Para instrumentar automaticamente rotas HTTP com o framework Fiber:

```go
package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/crewhu/observability_go/pkg/tracing/middleware"
)

func NewFiberHttp() *fiber.App {
	app := fiber.New()
	
	// Adiciona middleware de tracing
	app.Use(middleware.OtelMiddleware())
	
	// Configurações adicionais do Fiber...
	
	return app
}
```

### Registrando Logs

```go
import (
	"context"
	logging "github.com/crewhu/observability_go/pkg/logging"
)

func minhaFuncao(ctx context.Context) {
	// Diferentes níveis de log com contexto de tracing
	logging.Debug(ctx, "Detalhes de debug: %s", "informação detalhada")
	logging.Info(ctx, "Operação concluída com sucesso")
	logging.Warn(ctx, "Atenção: recurso com utilização alta")
	logging.Error(ctx, "Erro ao processar requisição: %v", err)
}
```

## Exemplos de uso do Tracing

### Criando spans, atributos e eventos

```go
import (
    "context"
    "github.com/crewhu/crewhu-trends-api/modules/analytics/src/infra/observability/tracing"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func exemploDeTracing(ctx context.Context, companyID, dataSourceID string) {
    // Inicia um novo span
    _, span := tracing.GetTracer("ListAvailableMetricsBySource").Start(ctx, "ListAvailableMetricsBySource")
    defer span.End()

    // Adiciona atributos ao span
    span.SetAttributes(
        attribute.String("company_id", companyID),
        attribute.String("data_source_id", dataSourceID),
    )

    // Adiciona um evento ao span
    span.AddEvent("metrics-found", trace.WithAttributes(
        attribute.String("list", "[lista de métricas aqui]"),
    ))

    // ... lógica da função ...
}
```

## Boas práticas para spans, atributos e eventos

### Nome dos Spans
- Use nomes descritivos e consistentes, preferencialmente no padrão `RecursoAção` (ex: `UserLogin`, `ListAvailableMetricsBySource`).
- O nome do span deve indicar claramente a operação ou endpoint monitorado.
- Para handlers HTTP, utilize o nome do caso de uso ou do controller.

### Quando utilizar Atributos (`SetAttributes`)
- Sempre que precisar registrar informações relevantes para análise e filtragem no trace.
- Exemplos de atributos:
  - IDs de entidades (`company_id`, `user_id`, `data_source_id`)
  - Parâmetros de entrada relevantes
  - Status ou resultado de operações
- Prefira atributos para dados que mudam pouco durante o span e são úteis para busca/agrupamento.

### Quando utilizar Eventos (`AddEvent`)
- Use eventos para registrar fatos importantes ou marcos dentro do span.
- Exemplos:
  - Resultado de uma consulta ou processamento (`metrics-found`, `validation-error`)
  - Mudanças de estado relevantes
  - Erros ou exceções capturadas durante a execução
- Eventos são ideais para registrar informações pontuais e detalhadas, que ajudam a entender o fluxo do span.

## Exemplo Completo de Integração

Veja um exemplo real de integração no repositório [crewhu-trends-api PR #323](https://github.com/crewhu/crewhu-trends-api/pull/323).

## Versionamento

Este projeto segue [Versionamento Semântico 2.0.0](https://semver.org/lang/pt-BR/):

- **MAJOR**: Alterações incompatíveis com versões anteriores
- **MINOR**: Adição de funcionalidades mantendo compatibilidade
- **PATCH**: Correções de bugs mantendo compatibilidade

Para mais informações sobre o processo de versionamento, consulte o [guia de versionamento](./docs/guides/versioning.md).

## Release Automatizada

A biblioteca implementa um processo de CI/CD que analisa as mensagens de commit para determinar automaticamente o tipo de versão a ser lançada:

- `feat(major):` ou mensagens com `BREAKING CHANGE` incrementam a versão MAJOR
- `feat:` incrementa a versão MINOR
- Outros tipos (`fix`, `docs`, etc.) incrementam a versão PATCH
