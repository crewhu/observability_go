# Guia de Release Automatizado

Este documento descreve como funciona o sistema automatizado de release para o projeto observability_go.

## Visão Geral

O processo de release foi completamente automatizado, permitindo que novas versões sejam criadas automaticamente quando código é enviado para a branch `main`. Este sistema:

1. Analisa o tipo de alterações feitas
2. Determina o incremento de versão adequado
3. Executa testes e verificações de qualidade
4. Gera notas de release com as alterações implementadas
5. Publica uma nova versão no GitHub e pkg.go.dev

## Fluxo de Trabalho

### 1. Desenvolvimento em Branch Feature

Sempre desenvolva novas funcionalidades em branches separadas:

```bash
git checkout -b feature/nova-funcionalidade
```

### 2. Commits com Mensagens Padronizadas

**IMPORTANTE**: Use as convenções de commit para categorizar suas alterações:

- `feat: nova funcionalidade` - Para novas funcionalidades (incrementa MINOR)
- `fix: correção de bug` - Para correções de bugs (incrementa PATCH)
- `docs: atualização na documentação` - Para mudanças na documentação
- `test: novos testes` - Para adição/modificação de testes
- `refactor: melhoria no código` - Para refatorações
- `perf: melhoria de performance` - Para otimizações
- `chore: atualização de dependências` - Para manutenção

Para quebras de compatibilidade, use um dos formatos:
- `feat(major): nova API incompatível`
- Qualquer commit com `BREAKING CHANGE:` no corpo da mensagem

### 3. Pull Request para Main

Quando sua feature estiver pronta:

1. Certifique-se que todos os testes passam
2. Crie um PR para a branch `main`
3. Aguarde a revisão e aprovação do PR

### 4. Merge e Release Automático

Após o merge para `main`:

1. O workflow de auto-release será acionado automaticamente
2. Uma nova versão será gerada baseada nos commits
3. Um changelog será produzido com as alterações
4. O release será publicado no GitHub
5. A nova versão estará disponível via `go get`

## Regras de Qualidade

O processo de release automatizado inclui verificações de qualidade:

- **Testes**: Todos os testes devem passar
- **Cobertura**: Mínimo de 75% de cobertura de código
- **Linting**: O código deve seguir os padrões definidos

Se alguma verificação falhar, o release não será criado.

## Exemplos Práticos

### Exemplo 1: Correção de Bug

```bash
git checkout -b fix/memory-leak
# Faça suas alterações
git add .
git commit -m "fix: corrige vazamento de memória no timer"
git push origin fix/memory-leak
# Crie um PR e aguarde aprovação
```

Resultado após merge: nova versão `vX.Y.Z+1` (incremento PATCH)

### Exemplo 2: Nova Funcionalidade

```bash
git checkout -b feature/opentelemetry-integration
# Implemente a nova feature
git add .
git commit -m "feat: adiciona integração com OpenTelemetry"
git push origin feature/opentelemetry-integration
# Crie um PR e aguarde aprovação
```

Resultado após merge: nova versão `vX.Y+1.0` (incremento MINOR)

### Exemplo 3: Mudança com Quebra de Compatibilidade

```bash
git checkout -b feature/new-api
# Implemente as alterações
git add .
git commit -m "feat(major): nova API de métricas"
git push origin feature/new-api
# Crie um PR e aguarde aprovação
```

Resultado após merge: nova versão `vX+1.0.0` (incremento MAJOR)
