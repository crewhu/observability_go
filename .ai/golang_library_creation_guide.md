# Guia Completo: Criando uma Biblioteca Go + GitHub - Do Zero ao Deploy

Este guia documenta o processo completo para criar uma biblioteca em Go, usando o GitHub como plataforma de hospedagem e distribuição, desde a concepção inicial até o deploy automatizado. Baseado na experiência do projeto `observability_go`.

## 1. Concepção e Planejamento

### 1.1 Definição do Escopo
- Defina claramente o problema que sua biblioteca resolverá
- Estabeleça os limites da sua solução (o que está dentro e fora do escopo)
- Identifique os usuários-alvo da biblioteca

### 1.2 Pesquisa e Referências
- Analise bibliotecas semelhantes existentes
- Identifique padrões comuns e melhores práticas
- Avalie pontos fortes e fracos das soluções existentes

### 1.3 Arquitetura e Design
- Defina a arquitetura da biblioteca (recomendado: Clean Architecture)
- Estabeleça os princípios de design (ex: simplicidade, extensibilidade)
- Esboce a API pública e os principais componentes

## 2. Configuração do Ambiente

### 2.1 Repositório no GitHub
```bash
# Crie um novo repositório no GitHub
# Copie a URL do repositório (exemplo: https://github.com/usuario/minha-lib.git)

# Clone localmente
git clone https://github.com/usuario/minha-lib.git
cd minha-lib
```

### 2.2 Inicialização do Módulo Go
```bash
# Inicialize o módulo Go com o nome completo do repositório
go mod init github.com/usuario/minha-lib
```

### 2.3 Configuração do .gitignore
```bash
# Crie um .gitignore específico para Go
cat > .gitignore << 'EOL'
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# Go workspace file
go.work

# IDE directories
.idea/
.vscode/
*.swp
*.swo

# OS specific files
.DS_Store
EOL
```

## 3. Estrutura do Projeto

### 3.1 Criação da Estrutura de Diretórios
```bash
# Crie a estrutura básica do projeto
mkdir -p pkg cmd internal docs examples scripts
```

### 3.2 Estrutura Recomendada (Clean Architecture)
```
minha-lib/
├── cmd/                  # Aplicações de linha de comando (se houver)
├── docs/                 # Documentação detalhada
│   └── guides/           # Guias de uso
├── examples/             # Exemplos práticos de uso
├── internal/             # Código interno, não exportado
├── pkg/                  # Código público, API exportada
├── scripts/              # Scripts de automação
├── .github/              # Configurações do GitHub
│   └── workflows/        # GitHub Actions
├── go.mod                # Definição do módulo Go
├── go.sum                # Checksums das dependências
├── README.md             # Documentação principal
└── LICENSE               # Licença do projeto
```

## 4. Implementação da Funcionalidade Core

### 4.1 Desenvolvimento da API Pública
- Crie os pacotes públicos em `pkg/`
- Implemente a funcionalidade core com interfaces claras
- Siga os princípios de design estabelecidos

### 4.2 Melhores Práticas de Implementação
- Mantenha a API pública mínima e bem documentada
- Encapsule detalhes de implementação em `internal/`
- Use interfaces para abstrair implementações concretas
- Garanta thread-safety quando necessário
- Siga as convenções de código Go

### 4.3 Documentação Inline
- Documente todos os elementos públicos com comentários godoc
- Inclua exemplos de uso nos comentários
- Adicione explicações sobre parâmetros e retornos

## 5. Testes Automatizados

### 5.1 Testes Unitários
```bash
# Crie arquivos de teste para cada pacote
touch pkg/meu_pacote/meu_pacote_test.go
```

### 5.2 Execução de Testes
```bash
# Execute todos os testes
go test ./...

# Execute testes com cobertura
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 5.3 Benchmarks
```go
// Adicione benchmarks aos seus testes
func BenchmarkMinhaFuncao(b *testing.B) {
    for i := 0; i < b.N; i++ {
        MinhaFuncao()
    }
}
```

## 6. Documentação

### 6.1 README.md
- Título e descrição clara da biblioteca
- Instruções de instalação e uso básico
- Exemplos de código
- Referência à documentação detalhada
- Informações sobre contribuição e licença

### 6.2 Exemplos Práticos
- Crie exemplos funcionais em `examples/`
- Demonstre casos de uso comuns
- Inclua exemplos avançados quando apropriado

### 6.3 Guias e Documentação Detalhada
- Crie documentação abrangente em `docs/`
- Inclua guias para casos de uso específicos
- Documente decisões de design e arquitetura

## 7. Configuração de CI/CD

### 7.1 GitHub Actions para Testes
```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run tests
        run: go test -race -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

### 7.2 Linting e Validação de Código
```yaml
# Adicione linting ao seu workflow
- name: golangci-lint
  uses: golangci/golangci-lint-action@v3
  with:
    version: latest
```

### 7.3 Workflow de Release Automatizado
```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run tests
        run: go test ./...
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          generate_release_notes: true
```

## 8. Versionamento e Publicação

### 8.1 Versionamento Semântico
- Adote o [Versionamento Semântico](https://semver.org/lang/pt-BR/)
- Defina convenções claras de versionamento no seu projeto

### 8.2 Tags e Releases Git
```bash
# Criando uma tag para uma nova versão
git tag -a v1.0.0 -m "Versão inicial estável"
git push origin v1.0.0
```

### 8.3 Publicação no pkg.go.dev
- O pkg.go.dev indexa automaticamente seu código quando:
  - O repositório é público no GitHub
  - Existe pelo menos uma tag de versão
  - Alguém executa `go get github.com/usuario/minha-lib`

### 8.4 Release Automatizado (Opcional)
- Use GoReleaser para automatizar o processo de release
- Configure um workflow para criar releases no GitHub
- Gere changelogs automaticamente baseados nos commits

## 9. Release Automatizado Avançado

### 9.1 Configuração do GoReleaser
```yaml
# .goreleaser.yml
before:
  hooks:
    - go mod tidy
builds:
  - skip: true  # Para bibliotecas, não precisamos gerar binários
archives:
  - format: tar.gz
    name_template: >-
      {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}
checksum:
  name_template: 'checksums.txt'
changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
```

### 9.2 Release Automatizado com Conventional Commits
- Utilize [Conventional Commits](https://www.conventionalcommits.org/pt-br/) para automatizar o controle de versão
- Configure ferramentas como commitlint para validar mensagens de commit
- Implemente workflows que criam novas versões baseadas no tipo de mudança

## 10. Manutenção Contínua

### 10.1 Gerenciamento de Issues
- Configure templates para issues no GitHub
- Categorize as issues com labels apropriados
- Responda rapidamente a bugs e perguntas

### 10.2 Contribuições Externas
- Estabeleça guias claros para contribuição
- Revise PRs de forma construtiva
- Mantenha a comunidade engajada

### 10.3 Evolução da Biblioteca
- Planeje novas funcionalidades com base no feedback
- Mantenha a compatibilidade com versões anteriores
- Documente mudanças importantes no CHANGELOG

## Recursos Adicionais

### Ferramentas Recomendadas
- **golangci-lint**: Linting e análise estática de código
- **GoReleaser**: Automação de releases
- **Codecov**: Análise de cobertura de código
- **go-critic**: Análise de qualidade de código
- **godoc**: Geração de documentação
- **commitlint**: Validação de mensagens de commit

### Links Úteis
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [GoReleaser Documentation](https://goreleaser.com/)
- [pkg.go.dev](https://pkg.go.dev/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)

## Exemplo Prático: observability_go

O projeto `observability_go` seguiu este guia para criar uma biblioteca de observabilidade em Go:

1. **Definição do escopo**: Biblioteca para medição de tempo de execução de funções
2. **Estrutura Clean Architecture**: Separação entre API pública e implementações internas
3. **Implementação core**: Funções de medição de tempo thread-safe e de baixo overhead
4. **Testes automatizados**: Cobertura >80% e benchmarks
5. **CI/CD**: GitHub Actions para testes, linting e releases automatizados
6. **Versionamento**: Release automatizado baseado em Conventional Commits
7. **Documentação**: README detalhado, exemplos práticos e guias de uso

Este processo resultou em uma biblioteca sólida, bem documentada e facilmente utilizável por outros desenvolvedores.
