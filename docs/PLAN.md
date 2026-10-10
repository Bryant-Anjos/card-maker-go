# Card Maker Go — Plano de implementação

> Status: **planejado, aguardando início da fase 1**. Nenhum código foi escrito ainda.

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
  - Pendente para a fase 5: conferir que os arquivos de arte das cartas existem e têm proporção compatível; o tema `default` só carrega depois da fase 4, quando seus assets existirem.
- [ ] Fase 4 — Renderer (design visual)
- [ ] Fase 5 — Pipeline em lote
- [ ] Fase 6 — Docs
- [ ] Fase 7 — API (opcional)

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
