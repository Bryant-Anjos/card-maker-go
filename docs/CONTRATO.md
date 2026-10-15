# Contrato de dados

Este documento define o que o Card Maker Go **lê** (cartas e tema) e o que ele **escreve** (imagens e
`manifest.json`). É a única coisa que outros projetos precisam conhecer: o programa é um detalhe de
implementação, o contrato é o que fica.

As definições formais estão em [`schema/`](https://github.com/Bryant-Anjos/card-maker-go/tree/main/schema) (`cards.schema.json`, `theme.schema.json`,
`manifest.schema.json` e `defs.schema.json`, que guarda as definições compartilhadas) (JSON Schema, rascunho 2020-12) e são embutidas
no binário (`schema.FS`). Um teste garante que os schemas e a validação em Go aceitam e rejeitam
exatamente os mesmos documentos.

## Regras que valem para todos os arquivos

| Regra | Detalhe |
|---|---|
| `schemaVersion` | Obrigatório em cartas, tema e manifest. Hoje é sempre `1`. |
| `$schema` | Opcional, só ajuda o editor (autocompletar). O programa ignora. Ex.: `"$schema": "../schema/cards.schema.json"`. |
| Campos desconhecidos | **Rejeitados.** Um erro de digitação falha na hora em vez de ser ignorado. |
| Identificadores | `id`, `rarity` e `element` seguem `^[a-z0-9][a-z0-9_-]*$` (viram nomes de arquivo). |
| Caminhos | Relativos, com `/`, sem segmentos vazios, `.` ou `..`, sem `\` e nunca absolutos. Um dado de entrada não consegue apontar para fora do diretório raiz. |
| Locale | Código como `pt-BR` ou `en` (`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`). |
| Texto localizado | Objeto `{ "pt-BR": "...", "en": "..." }`. |

## Entrada 1: cartas (`cards.json`)

```json
{
  "$schema": "../schema/cards.schema.json",
  "schemaVersion": 1,
  "cards": [
    {
      "id": "vulpin",
      "number": 1,
      "rarity": "common",
      "element": "beast",
      "name": { "pt-BR": "Vulpin" },
      "description": { "pt-BR": "Pequena criatura curiosa que vive em florestas tranquilas." },
      "art": "art/vulpin.png"
    }
  ]
}
```

| Campo | Obrigatório | Descrição |
|---|---|---|
| `id` | sim | Identificador único da carta. Vira o nome dos arquivos gerados. |
| `number` | não | Número da coleção (inteiro ≥ 0). Omitido ou `0` significa sem número. |
| `rarity` | sim | Id de uma raridade **definida no tema**. |
| `element` | não | Id de um elemento **definido no tema**. |
| `variant` | não | Id de uma variante de layout **definida no tema** (ex.: `fullart`). Omitido = layout padrão. |
| `name` | sim | Texto localizado, com ao menos um texto não vazio. |
| `description` | não | Texto localizado. |
| `art` | sim | Caminho da ilustração **relativo à pasta do `cards.json`**. PNG, JPEG ou WebP. |

Validações além do schema: ids únicos, raridade, elemento e variante existentes no tema, arte existente
e decodificável, e `name` disponível no locale pedido (ou no locale de fallback).

**Variantes** (como full art) são cartas próprias na coleção: cada versão tem seu `id`, sua arte e,
normalmente, seu `number`. Exemplo de uma versão full art do mesmo personagem:

```json
{ "id": "lunaris-fullart", "number": 132, "rarity": "rare", "element": "dark", "variant": "fullart",
  "name": { "pt-BR": "Lunaris" }, "art": "art/lunaris-fullart.png" }
```

## Entrada 2: tema (`theme.json`)

O tema é o que torna o programa agnóstico: **raridades e elementos são dados**, não código. Outro
jogo pode ter 3 ou 8 raridades sem tocar no programa.

```json
{
  "$schema": "../../schema/theme.schema.json",
  "schemaVersion": 1,
  "name": "default",
  "size": { "width": 750, "height": 1050 },
  "template": "card.html.tmpl",
  "stylesheet": "style.css",
  "back": "back.html.tmpl",
  "rarities": [
    { "id": "common", "label": { "pt-BR": "Comum" }, "color": "#8b9099", "frame": "frames/common.svg" }
  ],
  "elements": [
    { "id": "fire", "label": { "pt-BR": "Fogo" }, "icon": "icons/fire.svg" }
  ],
  "variants": [
    { "id": "fullart", "label": { "pt-BR": "Arte completa" }, "template": "fullart.html.tmpl" }
  ]
}
```

| Campo | Obrigatório | Descrição |
|---|---|---|
| `name` | sim | Nome do tema; aparece no manifest. |
| `size` | sim | Tamanho base da carta em pixels (`width`, `height`, > 0). O padrão do projeto é 750×1050 (proporção 5:7). |
| `template` | sim | Template HTML da carta, relativo à pasta do tema. |
| `stylesheet` | sim | CSS do tema. |
| `back` | não | Template do verso. Sem ele, nenhum verso é gerado. |
| `rarities[]` | sim (≥ 1) | `id`, `color` (`#rrggbb`), `frame` (arquivo) e `label` opcional. |
| `elements[]` | não | `id`, `icon` (arquivo) e `label` opcional. |
| `variants[]` | não | Layouts alternativos: `id`, `template` (arquivo) e `label` opcional. Cartas sem `variant` usam `template`. |

Todos os arquivos citados precisam existir na pasta do tema; o carregamento lista de uma vez todos
os que faltam. Os dados disponíveis dentro dos templates e as regras de renderização estão no
[guia de temas](TEMAS.md).

## Saída: imagens e `manifest.json`

Cada execução escreve em `<out>/<locale>/`:

```
out/pt-BR/
├── manifest.json
├── back.png  back@2x.png  back.webp  back@2x.webp
├── cards/
│   ├── vulpin.png  vulpin@2x.png  vulpin.webp  vulpin@2x.webp
│   └── ...
└── .cache.json            # cache do build; pode ser ignorado
```

- `<id>.<ext>` é a escala 1x; escalas maiores levam o sufixo `@<n>x` (`vulpin@2x.png`).
- O tamanho em pixels é sempre `size × escala`.
- Quem consome **não** deve montar caminhos por convenção: use o `path` de cada imagem no manifest.
- Com `cardmaker build -hashed-names`, o nome leva o hash do conteúdo antes da extensão
  (`cards/vulpin@2x.0c5d9e11.webp`, `back.3fa9c1d2.webp`) e o `path` do manifest é esse nome. O formato do
  `path` não muda (continua um caminho relativo), então o schema é o mesmo. Veja [CLI.md](CLI.md#nomes-com-hash).

```json
{
  "schemaVersion": 1,
  "theme": "default",
  "locale": "pt-BR",
  "size": { "width": 750, "height": 1050 },
  "back": [
    { "scale": 1, "format": "png", "path": "back.png", "width": 750, "height": 1050 }
  ],
  "cards": [
    {
      "id": "vulpin",
      "number": 1,
      "rarity": "common",
      "element": "beast",
      "name": "Vulpin",
      "description": "Pequena criatura curiosa que vive em florestas tranquilas.",
      "images": [
        { "scale": 1, "format": "png", "path": "cards/vulpin.png", "width": 750, "height": 1050 },
        { "scale": 2, "format": "webp", "path": "cards/vulpin@2x.webp", "width": 1500, "height": 2100 }
      ]
    }
  ]
}
```

| Campo | Descrição |
|---|---|
| `theme`, `locale`, `size` | De que tema, idioma e tamanho base vieram as imagens. |
| `cards[]` | Na **mesma ordem** do arquivo de cartas. |
| `cards[].name`, `description` | **Já resolvidos** para o locale: quem consome não precisa conhecer i18n. |
| `cards[].variant` | Variante de layout (ex.: `fullart`); ausente no layout padrão. |
| `cards[].images[]` | Uma entrada por escala e formato (`png` ou `webp`). |
| `back[]` | Imagens do verso; ausente se o tema não define verso. |

Os caminhos são relativos à pasta que contém o `manifest.json`, então a pasta inteira pode ser
copiada para um CDN, um bucket ou o diretório de assets de um app sem alterar nada.

## Versionamento

- Mudanças **compatíveis** (campo opcional novo) mantêm `schemaVersion: 1`.
- Mudanças que quebram quem consome incrementam `schemaVersion`. O programa recusa versões que não conhece
  com a mensagem `unsupported version N, expected M`, em vez de interpretar errado.
- Consumidores devem checar `schemaVersion` antes de ler o resto.

## Limitações atuais

- O WebP é codificado com **perda** por padrão (qualidade 90, escolhida por medição: PSNR acima de 40 dB e
  ~11x menor que o PNG em 2x; o canal alfa fica sempre exato). Qualidade 100 gera WebP sem perda, exato
  pixel a pixel, mas bem maior e mais lento. Veja `-webp-quality` no README.
- Ao remover uma carta dos dados, os arquivos antigos dela não são apagados da pasta de saída (o
  manifest deixa de listá-los).
- A proporção da arte não é validada; o template deve recortar a ilustração (`object-fit: cover`).
