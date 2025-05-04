# É um gol link - Biblioteca de Observabilidade

## Visão Geral
Biblioteca de infraestrutura Go para monitoramento e observabilidade, com foco inicial em medir o tempo de execução de funções arbitrárias.

## Objetivos
- Implementar uma interface simples para medir o tempo de execução de funções
- Estruturar o projeto seguindo Clean Architecture
- Disponibilizar como pacote versionado no GitHub
- Garantir código testável e de alta qualidade

## Implementação

### 1. Estrutura do Projeto
- [x] Criar repositório Git com estrutura padrão para bibliotecas Go
- [ ] Definir estrutura de diretórios conforme Clean Architecture
- [ ] Configurar `.gitignore` para Go
- [ ] Inicializar o módulo Go (go.mod)

### 2. Funcionalidade Core
- [ ] Desenvolver função de medição de tempo (relógio)
- [ ] Implementar interface simples e intuitiva
- [ ] Garantir thread-safety
- [ ] Adicionar documentação inline (godoc)

### 3. Testes
- [ ] Implementar testes unitários
- [ ] Configurar cobertura de código
- [ ] Adicionar benchmarks

### 4. Documentação
- [ ] Criar README.md com exemplos de uso
- [ ] Documentar API pública
- [ ] Adicionar exemplos práticos

### 5. CI/CD
- [ ] Configurar GitHub Actions para:
  - [ ] Testes automatizados
  - [ ] Linters (golangci-lint)
  - [ ] Publicação de novas versões

### 6. Versionamento
- [ ] Definir convenção de versionamento semântico
- [ ] Configurar tags Git
- [ ] Publicar no pkg.go.dev

## Considerações Técnicas
- Simplicidade: API minimalista e intuitiva
- Desempenho: Overhead mínimo nas medições
- Extensibilidade: Estrutura que permita adicionar recursos de observabilidade no futuro
- Compatibilidade: Integração com OpenTelemetry prevista

## Exemplo de Uso Futuro

```go
import "github.com/usuario/eumgollink"

func main() {
    // Medir tempo de execução de uma função
    resultado, duracao := eumgollink.TimeFunc(minhaFuncao)
    
    // Ou usar como decorator
    funcaoComMedicao := eumgollink.WithTiming(minhaFuncao)
    resultado := funcaoComMedicao()
}
```

## Próximas Fases
1. Implementação básica (medição de tempo)
2. Integração com OpenTelemetry
3. Exportação de métricas
4. Rastreamento de contexto