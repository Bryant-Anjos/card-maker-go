# Guia de integração

Como usar a saída do Card Maker Go em qualquer plataforma. O contrato completo está em
[CONTRATO.md](CONTRATO.md); aqui ficam receitas práticas.

## Princípio

O programa roda **uma vez, no build do conteúdo** (na sua máquina ou no CI), e gera uma pasta
autocontida. Os projetos (web, nativo, engine) só **leem o `manifest.json` e carregam imagens**; nenhum
deles reimplementa a composição da carta. Para trocar o visual, edita-se o tema e roda-se o build de novo.

```
cards.json + artes + tema ──► cardmaker build ──► out/pt-BR/ ──► web / app / engine
```

## Passo a passo

1. Gere as cartas: `cardmaker build -cards cards.json -theme themes/default -out out -locale pt-BR -scales 1,2,3 -formats png,webp`
2. Publique ou copie `out/pt-BR/` para onde o projeto lê assets (pasta do app, bucket, CDN).
3. No projeto, leia `manifest.json`, confira `schemaVersion` e use o `path` de cada imagem.

Escolhas úteis:

| Decisão | Recomendação |
|---|---|
| Escalas | `1,2` cobre web e a maioria dos celulares; acrescente `3` para telas muito densas. |
| Formatos | PNG funciona em todo lugar; WebP é opcional (lossless, veja as limitações no contrato). |
| Idiomas | Um diretório por locale, cada um com seu manifest. Troque de idioma trocando o diretório base. |
| Versionar a saída | Trate `out/` como artefato de build; publique no CDN com o conteúdo do jogo. |

## Web (JavaScript)

Testado com Node contra um manifest real gerado pelo programa.

```js
const base = "/assets/cards/pt-BR/";
const manifest = await (await fetch(`${base}manifest.json`)).json();

if (manifest.schemaVersion !== 1) throw new Error("manifest incompatível");

// srcset por formato: o navegador escolhe a escala conforme a densidade da tela
function srcset(card, format) {
  return card.images
    .filter((image) => image.format === format)
    .map((image) => `${base}${image.path} ${image.scale}x`)
    .join(", ");
}
```

```html
<picture>
  <source type="image/webp" srcset="/assets/cards/pt-BR/cards/vulpin.webp 1x, /assets/cards/pt-BR/cards/vulpin@2x.webp 2x">
  <img src="/assets/cards/pt-BR/cards/vulpin.png"
       srcset="/assets/cards/pt-BR/cards/vulpin.png 1x, /assets/cards/pt-BR/cards/vulpin@2x.png 2x"
       width="750" height="1050" alt="Vulpin">
</picture>
```

Dica: use `width`/`height` do manifest (tamanho base) para reservar espaço e evitar salto de layout.

## Nativo e engines (escolher a escala certa)

Apps nativos e engines costumam carregar **um** arquivo por carta. Esta função escolhe a menor escala que
atende à densidade da tela, ou a maior disponível:

```js
function pickImage(images, { format = "png", scale = 1 } = {}) {
  const candidates = images
    .filter((image) => image.format === format)
    .sort((a, b) => a.scale - b.scale);
  return candidates.find((image) => image.scale >= scale) ?? candidates.at(-1);
}
```

A lógica é a mesma em qualquer linguagem: filtrar por `format`, ordenar por `scale` e pegar a primeira
com `scale >= densidade`. O resto é ler JSON e carregar imagem com a API da plataforma:

| Plataforma | Como carregar |
|---|---|
| Flutter | `Image.asset`/`Image.network` com o `path`; decodifique o JSON com `dart:convert`. |
| React Native | `Image` com `source={{ uri }}` e `width`/`height` do manifest. |
| Swift / Kotlin | `Codable`/`kotlinx.serialization` para o manifest; imagens por URL ou bundle. |
| Godot / Unity | Carregue o JSON no startup; use `Texture2D`/`Sprite` a partir do `path`. |

## Go

Um programa em Go pode usar os mesmos tipos do pipeline (há um exemplo executável em
`pkg/cardmaker/model/example_test.go`):

```go
manifest, err := model.ParseManifest(file) // decodifica e valida
if err != nil { return err }

for _, card := range manifest.Cards {
    fmt.Println(card.Name, card.Images[0].Path)
}
```

## Boas práticas

- **Não construa caminhos por convenção**; use o `path` do manifest.
- **Ordem**: `cards[]` segue a ordem do arquivo de cartas, útil para listar a coleção por número.
- **Variantes**: `cards[].variant` (ex.: `fullart`) permite filtrar ou destacar versões especiais
  (`manifest.cards.filter((card) => card.variant === "fullart")`). A imagem já vem com o layout certo.
- **Cache no cliente**: os nomes dos arquivos não têm hash de conteúdo. Ao republicar, invalide o cache
  do CDN ou publique em um diretório novo por versão do conteúdo.
- **Validação no editor**: coloque `"$schema"` apontando para `schema/*.schema.json` nos seus arquivos de
  cartas e de tema para ter autocompletar e erros em tempo real.
- **Valide antes de publicar**: `cardmaker validate -cards cards.json -theme themes/default` confere dados,
  tema e artes sem renderizar nada, ideal para CI.

## API HTTP (cartas sob demanda)

Para gerar uma carta avulsa, como em uma ferramenta de edição ou pré-visualização, o `cardmaker-server`
expõe `POST /v1/render` sobre o mesmo núcleo: envie os dados da carta e a arte e receba a imagem. Veja o
[guia da API](API.md). Para a coleção estática, o fluxo de build acima continua sendo o caminho.
