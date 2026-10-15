# Referência da linha de comando

O projeto tem dois executáveis, gerados por `make build` em `bin/`:

- [`cardmaker`](#cardmaker): valida os dados e gera a coleção inteira (imagens + `manifest.json`).
- [`cardmaker-server`](#cardmaker-server): gera uma carta sob demanda por HTTP (rotas em [API.md](API.md)).

Todas as flags usam o formato `-nome valor` (ou `-nome` para as booleanas). Em caso de erro, as mensagens
dizem qual campo está errado e como corrigir; veja também a [solução de problemas](SOLUCAO-DE-PROBLEMAS.md).

## `cardmaker`

### `cardmaker version`

Imprime a versão (`0.1.0`).

### `cardmaker validate`

Confere o arquivo de cartas, o tema e as artes **sem renderizar nem escrever nada**: raridade, elemento e
variante de cada carta existem no tema, os textos existem no locale pedido, as artes existem e são imagens
válidas. Ideal para CI.

| Flag | Padrão | Descrição |
|---|---|---|
| `-cards` | (obrigatória) | Arquivo de cartas (`cards.json`); os caminhos de arte são relativos à pasta dele. |
| `-theme` | (obrigatória) | Pasta do tema (a que contém `theme.json`). |
| `-locale` | `pt-BR` | Idioma de saída. |
| `-fallback` | o próprio `-locale` | Idioma usado quando falta uma tradução. |

```bash
cardmaker validate -cards testdata/showcase/cards.json -theme themes/default
# OK: 5 cards are valid for theme "themes/default" and locale pt-BR
```

### `cardmaker build`

Renderiza todas as cartas (e o verso, se o tema tiver) e escreve `<out>/<locale>/manifest.json` mais as
imagens. Os dados são validados **antes** de qualquer renderização, e só o que mudou é renderizado de novo
(cache por hash; veja [ARQUITETURA.md](ARQUITETURA.md#cache-do-build)). Aceita todas as flags do `validate` e mais:

| Flag | Padrão | Descrição |
|---|---|---|
| `-out` | (obrigatória) | Pasta de saída; os arquivos vão para `<out>/<locale>/`. |
| `-scales` | `1,2` | Escalas das imagens, separadas por vírgula. A imagem tem `tamanho do tema × escala` pixels. |
| `-formats` | `png` | Formatos, separados por vírgula: `png`, `webp`. |
| `-webp-quality` | `90` | Qualidade do WebP: 1 a 99 com perda, `100` sem perda (exato, bem maior e mais lento). |
| `-concurrency` | `4` | Cartas renderizadas em paralelo. |
| `-force` | desligado | Ignora o cache e renderiza tudo de novo. |
| `-hashed-names` | desligado | Grava cada imagem como `<nome>.<hash>.<ext>` (veja abaixo) e apaga as imagens de gerações anteriores. |
| `-renderer` | `chrome` | `chrome` (real) ou `fake` (retângulos coloridos, sem navegador; útil para testar o pipeline). |
| `-chrome` | | Executável do navegador. Padrão: variável `CARDMAKER_CHROME`, depois o `PATH`. |
| `-no-sandbox` | `$CARDMAKER_CHROME_NO_SANDBOX` | Roda o navegador sem o sandbox. Só para contêineres (veja [DOCKER.md](DOCKER.md#o-sandbox-do-chromium)). |

```bash
cardmaker build -cards testdata/showcase/cards.json -theme themes/default \
  -out out -locale pt-BR -scales 1,2 -formats png,webp
# Built 5 cards (theme "default", locale pt-BR) in out/pt-BR: rendered 12, skipped 0
```

A segunda execução, sem mudanças, mostra `rendered 0, skipped 12`. O resumo conta imagens por escala
(5 cartas + verso, em 2 escalas = 12); os formatos saem da mesma renderização.

### Nomes com hash

Com a flag, cada imagem (cartas e verso, todas as escalas e formatos) é gravada com o hash do **conteúdo** no
nome, na mesma pasta: `back.3fa9c1d2.webp`, `back@2x.8b41e07a.webp`, `cards/vulpin.0c5d9e11.webp`. O hash são os
8 primeiros caracteres hexadecimais (minúsculos) do sha256 do arquivo. O `path` de cada imagem no
`manifest.json` passa a ser o nome com hash; os outros campos não mudam. Arte que mudou ganha uma URL nova e a
que não mudou mantém a mesma, então CDNs e caches de cliente que guardam por URL nunca entregam uma imagem velha.

- O cache do build continua valendo: o que não mudou não é renderizado de novo e o arquivo com hash existente
  é reaproveitado (os nomes ficam registrados em `.cache.json`).
- Ao fim do build, os arquivos de imagem (`.png` e `.webp`) de `<out>/<locale>/cards/` e os `back*` da raiz de
  `<out>/<locale>/` que **não estão no manifest novo** são apagados, para a pasta guardar uma geração só. Isso
  inclui os nomes sem hash de um build anterior. `manifest.json`, `.cache.json`, outros tipos de arquivo,
  subpastas e qualquer coisa fora de `<out>/<locale>/` nunca são tocados.
- Sem a flag nada muda e nada é apagado. Voltar a gerar sem a flag escreve os nomes sem hash e **não** apaga os
  com hash.
- Quem consome deve sempre ler o `path` do manifest (que, sem cache, é a única coisa a invalidar).
- O `cardmaker-server` não é afetado: ele renderiza uma imagem por pedido e não grava arquivos.

## `cardmaker-server`

Servidor HTTP para gerar cartas sob demanda. Rotas, segurança e limites estão em [API.md](API.md).

| Flag | Padrão | Descrição |
|---|---|---|
| `-addr` | `127.0.0.1:8080` | Endereço de escuta. |
| `-themes` | `themes` | Pasta cujas subpastas são temas; o **nome da subpasta** é o nome do tema na API. |
| `-renderer` | `chrome` | `chrome` ou `fake`. |
| `-chrome` | | Executável do navegador (mesma regra do `cardmaker build`). |
| `-no-sandbox` | `$CARDMAKER_CHROME_NO_SANDBOX` | Roda o navegador sem o sandbox; só para contêineres. |
| `-max-concurrent` | `4` | Renderizações em paralelo. |
| `-queue-timeout` | `10s` | Quanto uma requisição espera por uma vaga antes de receber `503`. |
| `-max-upload-mb` | `20` | Maior corpo de requisição aceito, em MiB. |
| `-allow-no-auth` | desligado | Permite um endereço não local sem token (veja a segurança em [API.md](API.md#segurança)). |

Ao subir, imprime `listening on http://<endereço>`; desliga de forma graciosa com `SIGINT`/`SIGTERM`.

```bash
CARDMAKER_API_TOKEN=um-token-longo bin/cardmaker-server -themes themes -addr 127.0.0.1:8080
```

## Variáveis de ambiente

| Variável | Usada por | Efeito |
|---|---|---|
| `CARDMAKER_CHROME` | os dois | Caminho do navegador, quando `-chrome` não é passado. |
| `CARDMAKER_CHROME_NO_SANDBOX` | os dois | `1`, `true`, `yes` ou `on` liga `-no-sandbox` por padrão. |
| `CARDMAKER_API_TOKEN` | `cardmaker-server` | Token exigido nas rotas (`Authorization: Bearer ...`). Vem do ambiente, e não de flag, para não aparecer na lista de processos. |

Como o navegador é encontrado: `-chrome`, depois `CARDMAKER_CHROME`, depois o `PATH`
(`chrome-headless-shell`, `chromium`, `chromium-browser`, `google-chrome`, `google-chrome-stable`,
`brave-browser`, `microsoft-edge`).

## Códigos de saída

| Código | Significado |
|---|---|
| `0` | Sucesso. |
| `1` | O comando rodou e falhou (dados inválidos, navegador ausente, erro de renderização...). A mensagem vai para o `stderr`. |
| `2` | Uso incorreto: comando ou flag desconhecidos, valor de flag mal formado. |
