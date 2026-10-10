# Card Maker Go — Plano de implementação

> Status: **concluído**. As 7 fases e os extras pedidos estão feitos (veja a seção "Progresso" no fim).
>
> Este é o **registro histórico** do planejamento e do progresso. Para saber como o programa funciona hoje,
> use a [documentação](README.md); este arquivo só é atualizado ao registrar entregas.

## 1. Objetivo

Um programa em Go que recebe **dados** e **imagens** de cartas e gera as imagens finais das cartas, para uso em um projeto de coleção de cartas (MVP; sem PvE/PvP planejados).

Motivação: evitar reescrever a lógica de geração de cartas a cada plataforma (web, nativo, engine). A geração fica em um programa separado e todos os projetos consomem a saída.

## 2. Princípios

- **Agnóstico**: o programa não sabe nada de raridades, elementos ou arte específicos. Tudo vem do **tema** e dos **dados**. Não gera arte, só compõe cartas.
- **Contrato, não código compartilhado**: a saída (imagens + `manifest.json` com schema documentado) é o que as outras plataformas consomem.
- **Núcleo em biblioteca**: CLI (e uma API futura) são cascas finas sobre `pkg/cardmaker`.
- **Multi-idioma na estrutura**, só `pt-BR` implementado no início.
- **TDD**, cobertura mínima de 80%.

## 3. Design de referência

A imagem [`reference/card-design-reference.png`](reference/card-design-reference.png) é a **base modelo** do design das cartas. O tema `default` deve reproduzi-la. Resumo do que ela define:

| Elemento | Descrição |
|---|---|
| Proporção da carta | Retrato, 5:7 (base 750×1050 px, configurável no tema) |
| Estrutura | Borda (varia por raridade), imagem da carta (arquivo), nome, descrição, número da coleção (canto superior direito da arte), ícone de elemento/categoria (ao lado do nome), elemento decorativo da raridade (losango na base) |
| Raridades | Comum (cinza), Incomum (verde), Raro (azul), Épico (roxo), Lendário (dourado). A borda ganha mais brilho e ornamento conforme a raridade |
| Elementos (exemplos) | Fogo, Água, Planta, Vento, Luz, Trevas (+ ícone de pata/garra, usado na carta Comum de exemplo) |
| Verso | Fundo azul-marinho com moldura dourada e rosa dos ventos central |
| Tipografia | Serifada para o nome da carta; descrição curta em uma ou duas linhas, leitura simples |
| Fundo da UI | Tons escuros azul-marinho/grafite |

Estes valores (raridades, elementos, cores) são **dados do tema `default`**, não regras do programa.

## 4. Decisões tomadas

| Tema | Decisão |
|---|---|
| Linguagem | Go |
| Estilização | HTML + CSS (template renderizado em navegador headless) |
| Plataforma-alvo | Indefinida, a saída precisa servir qualquer uma |
| Origem da arte | Irrelevante; apenas dados e imagens de teste em `testdata/` |
| Idiomas | Estrutura multi-idioma, só `pt-BR` no início |
| Dados de entrada | JSON (casa com JSON Schema) |
| Tamanho base | 750×1050 px, configurável por tema |
| Diretório | `card-maker-go` (kebab-case para não ter espaços no módulo Go e no shell) |

## 5. Opções consideradas

| Opção | Veredito |
|---|---|
| **A. Pré-renderizar com programa separado** | **Escolhida**: reuso total entre plataformas, clientes simples |
| B. Renderizar em cada cliente | Descartada: é o retrabalho que se quer evitar |
| C. Híbrido (PNG + efeitos no cliente) | Evolução futura: brilho/holo animados como overlay no cliente |

### Renderização HTML→imagem em Go

| Opção | Prós | Contras |
|---|---|---|
| **chromedp** (escolhida) | Padrão da comunidade, só Go | Exige Chrome/Chromium instalado |
| go-rod | API agradável, baixa o Chromium | Download automático menos previsível |
| Libs 2D puras (gg, canvas) | Sem Chrome | Sem CSS: perde glow, gradientes e tipografia |

## 6. Arquitetura

```
card-maker-go/
├── cmd/
│   ├── cardmaker/            # CLI: validate, render, build
│   └── cardmaker-server/     # API HTTP (fase 7, opcional)
├── pkg/cardmaker/            # núcleo reutilizável
│   ├── model/                # Card, Theme, Manifest (structs + validação)
│   ├── theme/                # carregar/validar temas
│   ├── render/               # HTML template → screenshot (interface Renderer)
│   ├── i18n/                 # resolução de textos por locale
│   └── manifest/             # geração do manifest
├── themes/default/
│   ├── theme.json            # raridades, elementos, cores, tamanho
│   ├── card.html.tmpl        # layout
│   ├── style.css
│   └── frames/ icons/ fonts/
├── schema/                   # JSON Schema: card, theme, manifest
├── docs/
│   ├── PLAN.md               # este documento
│   └── reference/            # design de referência
├── testdata/                 # cartas e artes placeholder
├── .tool-versions
└── go.mod
```

Pontos de design:

- **Raridade e elemento são dados do tema.** O `theme.json` declara `rarities: [{id, frame, color}]` e `elements: [...]`. Outro jogo pode ter 3 ou 8 raridades sem mexer no código.
- **Textos com locale**: `name: {"pt-BR": "Vulpin"}`, com fallback de idioma. Trocar de idioma só exige regerar.
- **Interface `Renderer`** isola o Chrome; o resto não depende dele.
- **Cache por hash** (dados + arte + tema + locale): só o que mudou é re-renderizado.
- **Saída**: `out/<locale>/cards/<id>.png`, variantes `@2x`, WebP opcional, `back.png` e `manifest.json`.
- **Versionamento**: `schemaVersion` no manifest e no tema desde o início.

## 7. Fases

1. **Fundação**: `go.mod`, `.tool-versions` (golang 1.27.1), estrutura de pastas, checagem local (`go vet`, `go test`), verificar o Chrome/Chromium.
2. **Contrato de dados (TDD)**: structs, validação e JSON Schemas de card, theme e manifest.
3. **Tema e i18n (TDD)**: loader de tema, resolução de locale com fallback, validação cruzada (a raridade da carta existe no tema).
4. **Renderer**: template HTML do tema `default` reproduzindo a referência e screenshot via chromedp. Testes com golden images e tolerância.
5. **Pipeline em lote**: CLI `build`, cache por hash, paralelismo controlado, 1x/2x/WebP, verso e manifest.
6. **Docs**: contrato (campos, versionamento, como consumir o manifest em web, nativo e engine) e README.
7. **(Opcional)** `cardmaker-server` com `POST /render`, só se surgir a necessidade.

MVP útil = fases 1 a 5.

## 8. Riscos

| # | Risco | Nível | Mitigação |
|---|---|---|---|
| 1 | Chrome ausente nesta máquina e dependência pesada | Alto | Verificar na fase 1; `Dockerfile`; erro claro; `Renderer` é interface |
| 2 | Golden tests instáveis entre máquinas (fontes, antialiasing) | Médio | Fontes embutidas no tema, tolerância na comparação, render em Docker |
| 3 | Over-engineering da API antes de precisar | Médio | API só na fase 7, opcional (YAGNI) |
| 4 | Schema mudar e quebrar consumidores | Médio | `schemaVersion` desde o início |
| 5 | Arte com proporção errada | Baixo | Validar dimensões; `object-fit: cover` |
| 6 | Efeitos animados (holo) não vão em PNG estático | Baixo | Overlay no cliente (opção C) |

**Complexidade geral:** média.

## 9. Observações do ambiente (2026-10-10)

- `go` instalado via asdf, sem versão definida globalmente; resolvido com `golang 1.27.1` no `.tool-versions` do projeto.
- Nenhum Chrome/Chromium no PATH, mas há binários utilizáveis nos caches do Playwright (`~/.cache/ms-playwright/chromium-*/chrome-linux64/chrome`) e do Puppeteer (`~/.cache/puppeteer/chrome/`), além do `brave-browser`. O renderer deve aceitar o caminho do Chrome por flag/variável de ambiente.

## 10. Progresso

- [x] **Fase 1 — Fundação**: `go.mod`, `.tool-versions`, estrutura de pastas, `Makefile` (`check`/`build`/`cover`), `.gitignore`, README, CLI mínimo (`cardmaker version`) e teste de versão. `make check` passa.
- [x] **Fase 2 — Contrato de dados**: pacote `pkg/cardmaker/model` (`Card`, `CardSet`, `Theme`, `Manifest`, `LocalizedText`) com validação que reporta todos os erros com o caminho do campo, parsing estrito (rejeita campos desconhecidos e dados após o JSON) e JSON Schemas em `schema/` (embutidos via `embed`). Um teste de conformidade garante que schema e validação Go aceitam e rejeitam os mesmos documentos. Dados de exemplo em `testdata/cards/cards.json`, `testdata/manifest/manifest.json` e `themes/default/theme.json`. Cobertura 97,5%.
- [x] **Fase 3 — Tema e i18n**: `theme.Load`/`theme.LoadDir` (lê `theme.json`, valida e confere que template, stylesheet, back, frames e ícones existem, reportando todos os arquivos faltantes de uma vez; carrega via `fs.FS`, então é testável sem disco), `CardSet.ValidateAgainst(theme)` (raridade e elemento da carta precisam existir no tema, com mensagem listando os válidos) e `i18n.Resolver` (resolve nome/descrição por locale com fallback e devolve `ResolvedCard`, o que o renderer vai consumir). Cobertura 97,9%.
  - O tema `default` passou a carregar com a fase 4.
- [x] **Fase 4 — Renderer e design visual** (feita pelo Opus, depois da 5 e da 6):
  - `render/chrome`: renderer real com chromedp v0.20. Executa o template do tema, serve página, arte e arquivos do tema por um servidor local (loopback, porta e token aleatórios) para que caminhos relativos funcionem, renderiza com `deviceScaleFactor` = escala e fundo transparente, espera fontes e imagens, e captura PNG. Uma aba por imagem, seguro para concorrência. `Version()` inclui a versão do navegador, então atualizar o Chrome invalida o cache.
  - Navegador: `-chrome`, `CARDMAKER_CHROME` ou `PATH`. Testado com `chrome-headless-shell` 151 e Brave 154.
  - Achado: `cdp.RGBA` omite alfa 0 (`omitzero`) e o fundo saía opaco; o renderer envia o comando com parâmetros próprios.
  - Tema `default` completo: template, CSS, moldura base, ornamentos por raridade (gerados por `internal/devtools/framegen`), 7 ícones, verso, fontes OFL embutidas. Detalhes em [TEMAS.md](TEMAS.md).
  - `testdata/showcase/`: artes recortadas da referência para revisão visual.
  - Regressão visual: golden images em `testdata/golden/default/`, comparação exata com o mesmo navegador (é determinístico: 0 pixels diferentes em 4 execuções) e tolerante com outros; uma troca de um dígito é detectada.
  - Diferença consciente: a referência tem proporção ≈ 3:5; o tema usa 5:7 (decisão do plano).
- [x] **Fase 5 — Pipeline em lote** (feita antes da 4, com renderer falso, a pedido): 
  - `render.Renderer` (interface: `Version`, `RenderCard`, `RenderBack`, sempre PNG com o tamanho do tema × escala) e `render.Fake` (retângulos na cor da raridade, sem navegador).
  - `pipeline.Run`/`pipeline.Validate`: valida tudo antes de renderizar (tema, cartas, raridades/elementos, textos do locale, arte existente e decodificável como png/jpeg/webp), renderiza em paralelo (`errgroup`), gera 1x/2x (`@2x`) em PNG e/ou WebP e grava `OutDir/<locale>/manifest.json` de forma atômica.
  - Cache por hash (versão do pipeline + versão do renderer + hash do tema inteiro + carta resolvida + hash da arte + escalas + formatos) em `.cache.json`; só re-renderiza o que mudou ou o que teve arquivo apagado; `-force` ignora o cache.
  - CLI: `cardmaker validate|build|version`; `-renderer chrome` (padrão, real desde a fase 4) ou `fake`.
  - Decisão de contrato: `manifest.back` virou **lista** de imagens (escalas e formatos), como as cartas.
  - Fixtures: tema de teste `testdata/themes/basic` (75×105 para os testes serem rápidos; o WebP em Go puro é lento em tamanho real) e artes placeholder em `testdata/cards/art` (`make testdata-art`).
  - Limitações conhecidas: WebP é **lossless** (encoder em Go puro); arquivos órfãos de cartas removidas não são apagados; a proporção da arte não é validada (o template deve usar `object-fit: cover`). Cobertura total 91,5%, sem races.
- [x] **Fase 6 — Docs**: [CONTRATO.md](CONTRATO.md) (entradas, saída, regras, versionamento, limitações) e [INTEGRACAO.md](INTEGRACAO.md) (receitas para web, nativo/engines e Go; os snippets JS foram executados contra um manifest real e o de Go é um `Example` verificado pelo `go test`). Ajuste de contrato: os arquivos de cartas, tema e manifest aceitam `"$schema"` opcional para autocompletar no editor. As variáveis do template foram documentadas na fase 4, em [TEMAS.md](TEMAS.md).
- [x] **Extra — Variante full art** (pedida após a aprovação dos exemplos): campo opcional `variant` nas cartas e no manifest, `variants` no tema (cada uma com template próprio), validação cruzada e schemas atualizados. Tema `default` ganhou `fullart.html.tmpl` (arte na carta inteira, brilho holográfico; após revisão do usuário, o texto fica direto sobre a arte, branco com contorno preto, sem painel) e a moldura base foi extraída para `frame.svg`, compartilhada pelos dois layouts. A refatoração alterou no máximo 4 pixels por carta, em 1 nível de cor (arredondamento), dentro da margem do golden estrito. Novos goldens: `lunaris-fullart` e `drakarion-fullart`.
- [x] **Fase 7 — API HTTP** (opcional, feita a pedido): `cmd/cardmaker-server` sobre o novo pacote `pkg/cardmaker/server`. Rotas: `GET /healthz`, `GET /v1/themes`, `POST /v1/render` (multipart: JSON da carta + arte; responde PNG/WebP, escala 1 a 3) e `GET /v1/themes/{name}/back`. Detalhes e exemplos em [API.md](API.md).
  - Segurança: escuta em localhost por padrão e **recusa endereço não local sem token** (`CARDMAKER_API_TOKEN`, comparação em tempo constante); limites de corpo (20 MiB), de pixels da arte e de escala; fila com no máximo `-max-concurrent` renderizações e `503 + Retry-After` quando lotada; erros internos genéricos para o cliente (detalhe só no log); panics recuperados; `nosniff`.
  - Respostas JSON usam o envelope `{success, data, error}`.
  - Refatorações de apoio: `pkg/cardmaker/imgenc` (codificação PNG/WebP e checagem de tamanho, antes privada do pipeline) e `pkg/cardmaker/renderers` (escolha do renderer por nome, agora compartilhada pelos dois executáveis).
  - Verificação de ponta a ponta com o binário real, Chrome e `curl`: carta padrão em 2x, full art em WebP, verso, erros, 12 requisições simultâneas, upload de 25 MB (413) e desligamento sem processos órfãos. A imagem da API é idêntica pixel a pixel à do CLI.
  - Bug encontrado ao testar o snippet JavaScript: `fetch`/`FormData` envia um `Blob` como parte de **arquivo**, e o servidor só aceitava o `card` como campo de texto. Corrigido (aceita os dois), com teste de regressão.
  - Fora do escopo: cache de imagens, TLS (use um proxy reverso), limite de taxa por cliente e CORS.
- [x] **Extra — WebP com compressão**: o WebP passou a ser **com perda por padrão (qualidade 90)**, com `-webp-quality` no CLI (1-99 com perda, 100 sem perda), `WebPQuality` no pipeline e `quality` na API. Trocado o encoder em Go puro (só sem perda) por `gen2brain/webp` (libwebp compilada para WASM e traduzida para Go: sem cgo nem biblioteca do sistema). Medido com o decodificador oficial do libwebp em 10 cartas reais: 2x cai de 20,5 MB (PNG) para 1,84 MB (**11x menor**), PSNR mínimo de 41,2 dB e canal alfa exato (cantos e brilho intactos). A qualidade entra na chave de cache. Concorrência verificada (32 codificações em paralelo, sem races, saída determinística, sem vazamento).
  - Achado: os decodificadores de WebP em Go (`x/image/webp` e o da própria `gen2brain/webp`) convertem a cor do WebP com perda de forma diferente do libwebp (faixa limitada BT.601 vs. faixa total), deslocando as cores em até ~9 níveis; uma primeira medição com eles dava falsos 28 dB. Documentado em INTEGRACAO.md para quem consome em Go.
- [x] **Extra — Docker**: `Dockerfile` multi-estágio (binários estáticos; Chromium do Debian; usuário 10001; `HEALTHCHECK`), `.dockerignore` e `docker-compose.yml` com `init`, sistema de arquivos somente leitura, `cap_drop: ALL`, `no-new-privileges`, limites de memória e processos e porta só em localhost. Novo `-no-sandbox` (e `CARDMAKER_CHROME_NO_SANDBOX`) nos dois executáveis, necessário em contêiner. Guia em [DOCKER.md](DOCKER.md).
  - Verificado de verdade com Docker 29: imagem de ~790 MB, build de ~2,5 min; recusa iniciar sem token; API respondendo (1,1 s por carta em 2x), saudável no healthcheck e com 401 sem token, tudo sob `--read-only --cap-drop ALL --security-opt no-new-privileges`; CLI com `--user` do host, volume somente leitura de entrada e cache funcionando (`rendered 0, skipped 12` na segunda execução); compose validado (e recusa subir sem o token). Render do contêiner vs. host: diferença média de 0,04/255.
  - Confirmado que o sandbox do Chromium não sobe no contêiner como usuário comum (`No usable sandbox!`), o que justifica a opção `-no-sandbox`; a troca de segurança está explicada em DOCKER.md.
  - Um `go mod download` falhou uma vez por erro de rede transitório (HTTP/2) e passou na segunda tentativa.
- [x] **Extra — Nome longo com fonte automática**: o renderer passou a aguardar `window.cardReady` (promessa opcional definida pelo tema; rejeição falha a renderização e promessa eterna estoura o timeout). O tema `default` ganhou `fit.js`, que encolhe elementos `data-fit` de 62 px até o mínimo `data-fit-min` (28 px) antes de recorrer às reticências; nenhum nome que já cabia é alterado (as 8 referências visuais anteriores seguem exatas). Mínimo calibrado por medição no pior caso (com ícone): 27 caracteres → 37 px, 31 → 32 px, 36 → 28 px, 58 → 17 px; um primeiro valor de 34 px ainda cortava nomes de 31 caracteres. Testes: o gancho (resolve, rejeita, ausente, timeout) e o comportamento real do ajuste medido no navegador (tamanho final e transbordamento) nos dois layouts, mais 6 novas referências visuais (`longnames.json`). A descrição continua limitada a 3 linhas com reticências.

## 11. Contrato de dados (resumo da fase 2)

- `schemaVersion` é obrigatório e hoje vale `1` em cards, theme e manifest.
- **Caminhos** (`art`, `frame`, `icon`, `template`, `stylesheet`, `back` e `images[].path`) são relativos, com `/`, sem segmentos vazios, `.` ou `..`, para que dados de entrada nunca apontem para fora do diretório raiz. O caminho de `art` é relativo à pasta do arquivo de cartas.
- **Identificadores** (`id`, `rarity`, `element`) seguem `^[a-z0-9][a-z0-9_-]*$`, pois viram nomes de arquivo.
- **Textos** são `LocalizedText` (`{"pt-BR": "..."}`); `name` exige ao menos um texto não vazio. A resolução tenta o locale exato, depois só o idioma (`en-US` → `en`) e depois o fallback.
- O **manifest** já traz os textos resolvidos para o locale, então quem consome não precisa conhecer i18n.
- Os JSON Schemas usam apenas regex portáveis (sem lookahead).

## 12. Critérios de aceite do MVP

- [ ] `cardmaker build` gera PNGs para todas as cartas de `testdata/` com o tema `default`
- [ ] O visual das 5 raridades e do verso é fiel à referência
- [ ] `manifest.json` válido contra o schema
- [ ] Validação aponta erros claros (raridade inexistente, arte ausente, locale faltando)
- [ ] Cache evita re-renderizar cartas inalteradas
- [ ] Cobertura de testes ≥ 80%
- [ ] `docs/` explica como outro projeto consome a saída
- [x] **Extra — README ilustrado e documentação viva**: README reescrito com imagens, diagramas Mermaid (validados com o mermaid.js) e início rápido executado de verdade; novos documentos `docs/README.md` (índice e política de atualização), `CLI.md` (referência completa), `ARQUITETURA.md`, `DESENVOLVIMENTO.md` e `SOLUCAO-DE-PROBLEMAS.md`, e imagens no guia de temas. As 6 imagens (`docs/images/`) são **geradas pela ferramenta real** (`make docs-images`, `internal/devtools/docimages`) a partir do tema `default` e dos dados de `testdata/showcase`, não desenhadas à mão.
  - Para a documentação não envelhecer: `internal/docscheck` (parte de `make check`; também `make docs-check`) falha se uma flag, subcomando, variável de ambiente, rota da API, pacote, schema ou alvo do `make` ficar sem documentação, se houver link ou âncora quebrada, documento fora do índice, imagem sem uso, ou se as imagens ficarem velhas em relação ao tema (`docs/images/SOURCE_HASH`). Verificado introduzindo 7 tipos de deriva, um por vez: todos foram pegos com a mensagem certa. `CLAUDE.md` repete as regras (mapa "mudou isto, atualize aquilo") para sessões futuras.
  - Achados: `pkg/cardmaker/model` não tinha comentário de pacote (adicionado); a tabela de flags da API duplicava a da CLI e passou a apontar para `CLI.md` (fonte única).
  - Instabilidade achada e corrigida: sob carga de CPU (8 núcleos ocupados) uma referência visual falhou 1 vez em 6 rodadas, com 45 pixels de texto variando até 3 níveis. O critério estrito era apertado demais (mais de 2 níveis). Agora um pixel só conta acima de 8 níveis e a diferença média deve ficar abaixo de 0,02, com testes do próprio critério (ruído passa; dígito trocado, pixel isolado, ajuste global de cor e mudança de alfa falham). Resultado: 0 falhas em 10 rodadas sob a mesma carga, e a troca de um dígito e um ajuste sutil de cor do painel continuam sendo pegos.
