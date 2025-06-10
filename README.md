# Observability Go

Biblioteca de infraestrutura Go para monitoramento e observabilidade, com foco inicial em medir o tempo de execução de funções arbitrárias.

## Instalação

```bash
go get github.com/crewhu/observability_go
```

Para usar uma versão específica:

```bash
go get github.com/crewhu/observability_go@v1.0.0
```

## Uso básico

### Medir tempo de execução de uma função

```go
import (
    "fmt"
    "github.com/crewhu/observability_go/pkg/timer"
)

func main() {
    // Medir tempo de execução de uma função
    resultado, duracao := timer.TimeFunc(func() string {
        // código a ser medido
        return "resultado"
    })
    
    fmt.Printf("Resultado: %s, Duração: %s\n", resultado, duracao)
}
```

### Decorar uma função com medição de tempo

```go
import (
    "fmt"
    "github.com/crewhu/observability_go/pkg/timer"
)

func main() {
    // Decorar função com medição de tempo e callback personalizado
    funcaoMedida := timer.WithTiming(
        func() int {
            // código a ser medido
            return 42
        },
        func(duracao time.Duration) {
            fmt.Printf("Função executada em: %s\n", duracao)
        },
    )
    
    resultado := funcaoMedida()
    fmt.Printf("Resultado: %d\n", resultado)
}
```

### Medir blocos de código específicos

```go
import "github.com/crewhu/observability_go/pkg/timer"

func minhaFuncao() {
    // Medir tempo do escopo inteiro
    defer timer.MeasureTime("operação completa")()
    
    // Código...
    
    {
        // Medir tempo de um bloco específico
        defer timer.MeasureTime("sub-operação")()
        // Código do bloco
    }
    
    // Mais código...
}
```

## Exemplos

Veja exemplos completos no diretório [examples](./examples).

## Versionamento

Este projeto segue [Versionamento Semântico 2.0.0](https://semver.org/lang/pt-BR/):

- **MAJOR**: Alterações incompatíveis com versões anteriores
- **MINOR**: Adição de funcionalidades mantendo compatibilidade
- **PATCH**: Correções de bugs mantendo compatibilidade

Para mais informações sobre o processo de versionamento, consulte o [guia de versionamento](./docs/guides/versioning.md).

## Contribuindo

Contribuições são bem-vindas! Sinta-se à vontade para abrir issues ou pull requests.

Para contribuir com o projeto:

1. Faça um fork do repositório
2. Crie uma branch para sua feature (`git checkout -b feature/nova-funcionalidade`)
3. Faça commit das suas alterações seguindo [Conventional Commits](https://www.conventionalcommits.org/pt-br/)
4. Envie um pull request

## Roadmap

1. ✅ Implementação básica (medição de tempo)
2. ⬜ Integração com OpenTelemetry
3. ⬜ Exportação de métricas
4. ⬜ Rastreamento de contexto

## Licença

MIT
