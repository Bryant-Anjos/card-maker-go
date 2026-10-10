# Desenvolvimento

Como preparar o ambiente, rodar os testes e contribuir. Para entender o código, comece pela
[arquitetura](ARQUITETURA.md).

## Preparando o ambiente

| Preciso de | Para | Observação |
|---|---|---|
| Go (versão em `.tool-versions`) | compilar e testar | `asdf install` lê o arquivo. |
| Um navegador Chromium | renderizar de verdade e rodar os testes visuais | Passe `CARDMAKER_CHROME=/caminho/do/navegador` ou deixe no `PATH`. Sem navegador, os testes que dependem dele são **pulados**, não falham. |
| Python 3 com Pillow e a fonte DejaVu Sans | só `make docs-images` | Não é necessário para compilar nem testar. |
| Docker | só para a imagem ([DOCKER.md](DOCKER.md)) | Opcional. |

```bash
git clone <repositório> && cd card-maker-go
export CARDMAKER_CHROME=/usr/bin/chromium   # ou o navegador que você tiver
make check build
bin/cardmaker version
```

## Comandos do `make`

| Comando | O que faz |
|---|---|
| `make build` | Compila `bin/cardmaker` e `bin/cardmaker-server`. |
| `make test` | `go test ./...`. |
| `make vet` | `go vet ./...`. |
| `make check` | `vet` + `test`. Rode antes de cada commit. |
| `make cover` | Testes com cobertura; imprime o total. A meta do projeto é 80% ou mais. |
| `make testdata-art` | Regera as artes de teste (degradês) de `testdata/cards/art`. |
| `make theme-frames` | Regera os ornamentos por raridade do tema `default` (`themes/default/frames`). |
| `make golden` | Regrava as imagens de referência dos testes visuais depois de uma mudança **intencional** no visual. |
| `make docs-images` | Re-renderiza as imagens da documentação com a ferramenta real e atualiza `docs/images/SOURCE_HASH`. |
| `make docs-check` | Roda só os testes que verificam a documentação. |

## Testes

O projeto segue TDD: **escreva o teste, veja falhar, implemente, veja passar**. Há testes unitários,
de integração (CLI, pipeline e API de ponta a ponta com o renderer falso), de comportamento no navegador real
e de regressão visual.

- **Cobertura**: meta de 80% ou mais (hoje, cerca de 92%). `make cover` mostra o total.
- **Race detector**: `go test -race ./...` precisa passar (o pipeline e a API usam concorrência). É mais lento
  por causa do WebP, que é código WebAssembly traduzido para Go.
- **Sem navegador**: `CARDMAKER_CHROME=/inexistente go test ./...` deve passar; os testes do navegador se pulam.
- **Imagens de referência (goldens)**: `pkg/cardmaker/render/chrome/golden_test.go` renderiza amostras com o
  tema `default` e compara com `testdata/golden/default/`. Com o **mesmo navegador** que gerou as imagens
  (registrado em `testdata/golden/default/RENDERER`) a comparação é estrita (ruído de até 8 níveis por pixel e média
  abaixo de 0,02, porque sob carga pesada o Chrome já variou até 3 níveis em poucos pixels): até um
  dígito trocado falha. Com outro navegador ou versão, só vale uma tolerância (o serrilhado do texto muda).
  Depois de uma mudança visual intencional, rode `make golden`, **olhe as imagens** e commite junto.
- **Conformidade do contrato**: `schema/schema_test.go` roda os mesmos documentos no JSON Schema e na validação
  em Go; se um lado mudar sem o outro, o teste falha.

## Convenções de código

- Arquivos pequenos e coesos (200-400 linhas, no máximo 800) e funções curtas (menos de 50 linhas).
- **Imutabilidade**: devolva valores novos em vez de alterar os recebidos.
- Trate todos os erros explicitamente, com o contexto de onde ocorreram; nunca engula um erro.
- Valide tudo que vem de fora (dados, parâmetros, uploads) na borda do sistema.
- Sem segredos no código; o token da API vem do ambiente.
- Nomes descritivos; constantes nomeadas no lugar de números mágicos.

## Commits

- Formato *conventional commits*, em inglês: `feat: ...`, `fix: ...`, `docs: ...`, `test: ...`, `refactor: ...`,
  `chore: ...`, com o porquê no corpo da mensagem.
- Mudanças de código e a documentação delas vão **no mesmo commit**.
- Commite o trabalho concluído antes de começar a próxima etapa.

## Documentação viva

A documentação precisa acompanhar o código. Regra simples: **quem muda algo que está documentado, ou que
deveria estar, atualiza a documentação na mesma mudança**. A tabela diz onde:

| Se você mudar... | Atualize |
|---|---|
| uma flag, subcomando ou variável de ambiente | [CLI.md](CLI.md) |
| uma rota, parâmetro, limite ou regra de segurança da API | [API.md](API.md) |
| o formato de cartas, tema ou manifest, ou um JSON Schema | [CONTRATO.md](CONTRATO.md) e os exemplos de `testdata/` |
| templates, CSS, fontes, ícones ou variantes do tema | [TEMAS.md](TEMAS.md), e rode `make golden` e `make docs-images` |
| pacotes (criar, mover, mudar responsabilidade) | [ARQUITETURA.md](ARQUITETURA.md) |
| `Dockerfile`, compose ou o sandbox | [DOCKER.md](DOCKER.md) |
| o consumo da saída (formatos, qualidade, receitas) | [INTEGRACAO.md](INTEGRACAO.md) |
| alvos do `make`, ferramentas ou o fluxo de desenvolvimento | este documento |
| um erro comum ou uma armadilha descoberta | [SOLUCAO-DE-PROBLEMAS.md](SOLUCAO-DE-PROBLEMAS.md) |
| uma entrega relevante | a seção de progresso do [PLAN.md](PLAN.md) |
| a visão geral, os recursos ou o início rápido | o [README](../README.md) |
| um documento novo | o índice [docs/README.md](README.md) |

Três mecanismos ajudam a não esquecer:

1. **Testes de documentação** (`internal/docscheck`, `make docs-check`, também parte de `make check`): falham
   se uma flag, subcomando, variável de ambiente, rota da API, pacote, schema ou alvo do `make` não está
   documentado; se um link ou âncora está quebrado; se um documento não está no índice; se há imagem sem uso.
   A mensagem de erro diz onde documentar.
2. **Impressão digital das imagens**: `docs/images/SOURCE_HASH` guarda o hash do tema `default` e dos dados de
   amostra de onde as imagens saíram. Mudou o tema? O teste falha até você rodar `make docs-images`.
3. **`CLAUDE.md`**: as instruções para o assistente de código repetem essa regra, para que ela valha também
   em sessões futuras.

Os testes só pegam o que é verificável por máquina (existência, links, hash). Se a **explicação** de algo mudou
de sentido, quem altera o código precisa revisar o texto, conferindo os exemplos: eles devem ser executados,
não só escritos.

## Ferramentas de desenvolvimento

- `internal/devtools/placeholderart`: gera as artes de teste (degradês) usadas em `testdata/cards/`.
- `internal/devtools/framegen/gen_frames.py`: gera os ornamentos SVG por raridade do tema `default`.
- `internal/devtools/docimages/make_images.py`: gera as imagens da documentação; as posições das chamadas da
  imagem de anatomia valem para o layout do tema `default` e precisam ser revistas se ele mudar.
- `internal/devtools/docsourcehash`: imprime o hash das fontes das imagens (usado pelo `make docs-images`).
