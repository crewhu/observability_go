# Observability Go

Biblioteca de infraestrutura Go para monitoramento e observabilidade, com foco inicial em medir o tempo de execução de funções arbitrárias.

## Instalação

```bash
go get github.com/iamviniciuss/observability_go
```

## Uso básico

### Medir tempo de execução de uma função

```go
import (
    "fmt"
    "github.com/iamviniciuss/observability_go/pkg/timer"
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
    "github.com/iamviniciuss/observability_go/pkg/timer"
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
import "github.com/iamviniciuss/observability_go/pkg/timer"

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

## Contribuindo

Contribuições são bem-vindas! Sinta-se à vontade para abrir issues ou pull requests.

## Roadmap

1. ✅ Implementação básica (medição de tempo)
2. ⬜ Integração com OpenTelemetry
3. ⬜ Exportação de métricas
4. ⬜ Rastreamento de contexto

## Licença

MIT
