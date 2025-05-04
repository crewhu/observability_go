# Guia de Versionamento

Este documento descreve as convenções de versionamento e o processo de release para a biblioteca observability_go.

## Versionamento Semântico

Este projeto segue estritamente as convenções de [Versionamento Semântico 2.0.0](https://semver.org/lang/pt-BR/).

Cada versão segue o formato: `MAJOR.MINOR.PATCH`

- **MAJOR**: Alterações incompatíveis com versões anteriores
- **MINOR**: Adição de funcionalidades mantendo compatibilidade
- **PATCH**: Correções de bugs mantendo compatibilidade

## Convenções de Commits

Para facilitar a geração de changelogs automáticos e ajudar na determinação do próximo número de versão, seguimos o padrão [Conventional Commits](https://www.conventionalcommits.org/pt-br/):

- `feat:` - Nova funcionalidade (incrementa MINOR)
- `fix:` - Correção de bug (incrementa PATCH)
- `docs:` - Alteração na documentação
- `style:` - Formatação que não afeta o código
- `refactor:` - Refatoração que não altera funcionalidade
- `perf:` - Melhoria de performance
- `test:` - Adição/correção de testes
- `chore:` - Alterações no processo de build, ferramentas, etc.

Exemplos:
```
feat: adiciona função para rastreamento de contexto
fix: corrige vazamento de memória na função TimeFunc
docs: melhora exemplos de uso da API
```

## Processo de Release

### Release Manual

1. **Preparação**:
   - Garantir que todos os testes estão passando
   - Verificar a cobertura de código
   - Atualizar documentação se necessário

2. **Criar uma tag para a versão**:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

3. **Release automática**:
   - O GitHub Actions irá detectar a nova tag
   - Os testes serão executados novamente
   - O GoReleaser irá criar a release no GitHub
   - O módulo será publicado no pkg.go.dev automaticamente

4. **Versões preliminares**:
   Para versions beta ou release candidates, use:
   ```bash
   git tag -a v1.0.0-beta.1 -m "Beta release v1.0.0-beta.1"
   git push origin v1.0.0-beta.1
   ```

### Release Automatizada (CI/CD)

Implementamos um processo automatizado de release que funciona da seguinte forma:

1. **Push para branch main**:
   - Ao enviar código para a branch `main`, o GitHub Actions dispara o workflow de auto-release

2. **Análise do commit**:
   - O sistema analisa automaticamente as mensagens de commit para determinar o tipo de versão:
     - `feat(major):` ou mensagens com `BREAKING CHANGE` incrementam a versão MAJOR
     - `feat:` incrementa a versão MINOR
     - Outros tipos (fix, docs, etc.) incrementam a versão PATCH

3. **Validação**:
   - Os testes são executados automaticamente
   - A cobertura de código é verificada (mínimo de 75%)
   - Se os testes falharem ou a cobertura for insuficiente, o release é abortado

4. **Geração de versão**:
   - Uma nova tag é criada automaticamente seguindo o versionamento semântico
   - O changelog é gerado com base nos commits desde a última versão
   - Um release é publicado no GitHub com as notas de mudança
   - O pacote é disponibilizado automaticamente no pkg.go.dev

Para que este processo funcione corretamente, é fundamental seguir as [Convenções de Commits](#convenções-de-commits) mencionadas acima.

## Atualização do go.mod

O arquivo go.mod será atualizado automaticamente com a nova versão quando os usuários instalarem a biblioteca. Não é necessário alterar manualmente este arquivo para cada release.

## Compatibilidade

- Garantimos compatibilidade com as 3 últimas versões do Go
- Mudanças que quebram compatibilidade (MAJOR) serão documentadas claramente
- Migrations ou guias de atualização serão fornecidos para versões com quebra de compatibilidade
