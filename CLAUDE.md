# Card Maker Go: instruções para o assistente de código

Gerador de cartas colecionáveis em Go: dados + ilustrações + tema → imagens (PNG/WebP) e `manifest.json`.
Há um CLI (`cmd/cardmaker`), uma API HTTP (`cmd/cardmaker-server`) e um tema de exemplo (`themes/default`).
Comece por `docs/README.md` (índice) e `docs/ARQUITETURA.md` (mapa do código).

## Comandos

- `make check`: `go vet` + todos os testes. **Rode antes de cada commit.** Inclui a verificação da documentação.
- `make build`: compila `bin/cardmaker` e `bin/cardmaker-server`.
- `make cover`: cobertura (meta: 80% ou mais).
- `make golden`: regrava as imagens de referência visuais. Só depois de uma mudança visual intencional.
- `make docs-images`: regera as imagens da documentação (precisa de um Chromium e do Pillow).
- Os testes de navegador usam o `chrome-headless-shell` do Playwright: exporte
  `CARDMAKER_CHROME=~/.cache/ms-playwright/chromium_headless_shell-*/chrome-headless-shell-linux64/chrome-headless-shell`
  para a comparação estrita com as imagens de referência (outro navegador só vale com tolerância). O critério
  tolera ruído de renderização (até 8 níveis por pixel, média abaixo de 0,02); não o afrouxe para "consertar"
  um teste visual que falha: investigue onde estão os pixels diferentes.

## Como trabalhar

- **TDD**: escreva o teste, veja falhar (RED), implemente (GREEN), refatore. Cobertura de 80% ou mais.
- **Imutabilidade**: devolva valores novos; não altere os recebidos.
- Arquivos pequenos e coesos, funções curtas, erros tratados com contexto, validação nas bordas.
- Não invente comportamento: **execute** os exemplos e comandos antes de documentá-los.
- Ao mexer em código que lida com upload, caminhos, autenticação ou execução de processos, revise a segurança
  (`docs/API.md`, seção Segurança, lista o que já é garantido).

## Commits

- Conventional commits, **em inglês** (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`), com o porquê no corpo.
- **Sem linhas `Co-Authored-By`**: foi um pedido explícito do usuário para este projeto.
- Commite o trabalho concluído **antes de começar a próxima etapa**, só com os arquivos dela.
- Código e a documentação dele vão no **mesmo commit**.

## Documentação viva (regra de ouro)

A documentação (em português) deve estar sempre em dia. **Ao alterar algo que está documentado, ou que deveria
estar, atualize a documentação na mesma mudança, antes de commitar.**

| Se mudou... | Atualize |
|---|---|
| flag, subcomando ou variável de ambiente | `docs/CLI.md` |
| rota, parâmetro, limite ou segurança da API | `docs/API.md` |
| formato de cartas/tema/manifest, JSON Schema | `docs/CONTRATO.md` + exemplos em `testdata/` |
| templates, CSS, fontes, ícones ou variantes do tema | `docs/TEMAS.md`, `make golden`, `make docs-images` |
| pacotes (criar, mover, mudar responsabilidade) | `docs/ARQUITETURA.md` |
| `Dockerfile`, compose ou sandbox | `docs/DOCKER.md` |
| consumo da saída (formatos, qualidade, receitas) | `docs/INTEGRACAO.md` |
| alvos do `make`, ferramentas, fluxo de desenvolvimento | `docs/DESENVOLVIMENTO.md` |
| erro comum ou armadilha descoberta | `docs/SOLUCAO-DE-PROBLEMAS.md` |
| entrega relevante | seção "Progresso" de `docs/PLAN.md` |
| visão geral, recursos, início rápido | `README.md` |
| documento novo | índice em `docs/README.md` **e** `nav` em `mkdocs.yml` |

Os testes de `internal/docscheck` falham se uma flag, rota, pacote, schema ou alvo do `make` ficar sem
documentação, se houver link ou âncora quebrada, documento fora do índice, imagem sem uso, ou se as imagens
ficarem velhas em relação ao tema (`docs/images/SOURCE_HASH`). A mensagem de erro diz onde documentar. Eles
não substituem o seu julgamento: se o **sentido** de algo mudou, revise o texto mesmo sem teste falhando.

O site da documentação (GitHub Pages) é montado de `docs/` com o MkDocs (`mkdocs.yml`, `make docs-site`); links
para fora de `docs/` usam a URL do GitHub. O CI (`.github/workflows/ci.yml`) roda `make check` equivalente.

Antes de commitar: `make check` verde, documentação atualizada, `make docs-images` se o tema `default` ou os
dados de `testdata/showcase` mudaram.

## Armadilhas conhecidas

- O hash do cache cobre **todos** os arquivos do tema; qualquer edição no tema re-renderiza tudo.
- Os decodificadores de WebP em Go deslocam a cor do WebP com perda; para medir qualidade, use o libwebp
  (por exemplo, via Pillow), e não `x/image/webp`.
- Em testes com Docker, não publique portas fixas: use `-p 127.0.0.1::8080` (a 8096 já foi de outro serviço).
- `make golden` regrava **todas** as imagens de referência; restaure com `git checkout` as que mudaram só por
  ruído de arredondamento (1 nível de cor) para o commit conter apenas o que realmente mudou.
