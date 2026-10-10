# Solução de problemas

Os sintomas abaixo aparecem com mensagens reais do programa. Se o seu caso não está aqui, rode primeiro
`cardmaker validate`, que aponta o campo exato de qualquer problema nos dados ou no tema.

## Navegador e renderização

| Sintoma | Causa e solução |
|---|---|
| `no Chrome/Chromium browser found in PATH; install one, pass -chrome <path> or set CARDMAKER_CHROME` | Não há navegador Chromium instalado, ou ele não está no `PATH`. Instale um (`chromium`, `google-chrome`, `brave-browser`...) ou informe o caminho com `-chrome` ou `CARDMAKER_CHROME`. Para testar o pipeline sem navegador: `-renderer fake`. |
| `No usable sandbox!` (dentro do Docker ou de outro contêiner) | O sandbox do Chromium precisa de *user namespaces*, que contêineres costumam bloquear. Passe `-no-sandbox` (ou `CARDMAKER_CHROME_NO_SANDBOX=1`); a imagem do projeto já faz isso. Entenda a troca de segurança em [DOCKER.md](DOCKER.md#o-sandbox-do-chromium). |
| A renderização demora e termina com `context deadline exceeded` | A página do tema não terminou de carregar: uma promessa `window.cardReady` que nunca resolve, ou um recurso remoto (fonte, imagem) inacessível. O limite é de 60 s por imagem. Veja o [guia de temas](TEMAS.md#scripts-do-tema-windowcardready). |
| Fontes diferentes das esperadas | O tema deve embutir as fontes com `@font-face` e arquivos locais; fontes do sistema variam entre máquinas. |

## Dados e tema

| Sintoma | Causa e solução |
|---|---|
| `rarities[0].frame: file "frames/common.svg" not found in theme` | O `theme.json` cita um arquivo que não existe na pasta do tema. Todos os arquivos faltantes são listados de uma vez. |
| `cards[2].rarity: unknown rarity "mythic", theme "default" defines: common, ...` | A carta usa uma raridade (ou elemento, ou variante) que o tema não define. A mensagem lista os valores válidos. |
| `card "x": art "art/x.png": is not a supported image (png, jpeg or webp)` | A arte não existe ou não é PNG, JPEG ou WebP. SVG não é aceito como arte. |
| `card "x": no name for locale "en" (fallback "pt-BR")` | A carta não tem nome no idioma pedido nem no de fallback. Acrescente a tradução ou ajuste `-fallback`. |
| `unsupported version 2, expected 1` | O arquivo usa um `schemaVersion` que esta versão do programa não conhece. |
| `decode card set: json: unknown field "..."` | Há um campo desconhecido (provavelmente um erro de digitação): campos desconhecidos são rejeitados de propósito. |

## Saída e cache

| Sintoma | Causa e solução |
|---|---|
| Mudei algo e a carta não atualizou | O cache considera dados, arte, tema, escalas, formatos e qualidade; se mesmo assim algo ficou velho, use `-force`. Apagar `<out>/<locale>/.cache.json` também refaz tudo. |
| Tudo é renderizado de novo sem eu ter mudado nada | Um arquivo do tema mudou (o hash cobre **todos** os arquivos da pasta) ou o navegador foi atualizado. É o comportamento esperado. |
| Ficaram arquivos de cartas que removi | Arquivos de cartas removidas não são apagados da saída; o manifest só deixa de listá-los. Apague a pasta de saída se quiser limpar. |
| Cores levemente diferentes ao decodificar o WebP em Go | Os decodificadores de WebP do ecossistema Go convertem a cor do WebP com perda de forma diferente do libwebp. Navegadores e celulares exibem corretamente; em Go, use o PNG. Veja [INTEGRACAO.md](INTEGRACAO.md). |

## API e Docker

| Sintoma | Causa e solução |
|---|---|
| `refusing to listen on "0.0.0.0:8080" without authentication` | O servidor se recusa a escutar fora de localhost sem token. Defina `CARDMAKER_API_TOKEN`, use um endereço local ou, só se souber o que faz, `-allow-no-auth`. |
| `401` | Token ausente ou errado: envie `Authorization: Bearer <token>`. |
| `503` com `Retry-After` | Todas as vagas de renderização estavam ocupadas além do `-queue-timeout`. Tente de novo, ou aumente `-max-concurrent` se a máquina aguentar. |
| `413` | O corpo passou de `-max-upload-mb` (20 MiB por padrão). |
| `port is already allocated` ao subir o contêiner | A porta do host já está em uso, talvez por outro serviço seu. Use `-p 127.0.0.1::8080` para o Docker escolher uma porta livre (veja com `docker port`). |
| O build da imagem falha em `go mod download` com `stream error ... INTERNAL_ERROR` | Erro de rede transitório com o proxy de módulos Go. Rode o build de novo; as camadas anteriores ficam em cache. |

## Testes e documentação

| Sintoma | Causa e solução |
|---|---|
| `N pixels differ by more than 8 levels ... from the golden` | O visual do tema mudou. Se foi intencional, `make golden`, **olhe as imagens** e commite junto da mudança. Se não foi, é uma regressão. |
| `goldens were made by ... comparing with tolerance only` | Os testes rodam com outro navegador ou versão que o das imagens de referência, então só valem com tolerância. Para comparação exata, use o navegador registrado em `testdata/golden/default/RENDERER`. |
| `the documentation images are out of date` | O tema `default` ou os dados de `testdata/showcase` mudaram. Rode `make docs-images` e commite `docs/images/`. |
| `... not documented in docs/...` | Algo novo no código (flag, rota, pacote, alvo do `make`...) ficou sem documentação. A mensagem diz onde documentar. Veja [DESENVOLVIMENTO.md](DESENVOLVIMENTO.md#documentação-viva). |
