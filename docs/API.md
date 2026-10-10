# API HTTP

O `cardmaker-server` gera **uma carta sob demanda** por HTTP, sobre o mesmo núcleo do CLI (mesmos temas,
mesmas validações, mesmos pixels: uma carta renderizada pela API é idêntica à gerada pelo `build`). É
pensado para ferramentas de edição e pré-visualização. Para gerar a coleção inteira, continue usando o
`cardmaker build` (veja [INTEGRACAO.md](INTEGRACAO.md)).

## Executando

```bash
make build   # gera bin/cardmaker e bin/cardmaker-server
CARDMAKER_API_TOKEN=um-token-longo bin/cardmaker-server -themes themes -addr 127.0.0.1:8080
```

Todas as flags (`-addr`, `-themes`, `-max-concurrent`, `-queue-timeout`...) e variáveis de ambiente estão em
[CLI.md](CLI.md#cardmaker-server).

O servidor imprime `listening on http://<endereço>` ao subir e desliga de forma graciosa com `Ctrl+C`
(`SIGINT`) ou `SIGTERM`, aguardando as requisições em andamento.

## Segurança

- **Escuta só em localhost por padrão.** Em um endereço não local (`0.0.0.0`, `:8080`, IP da rede), o
  servidor **se recusa a iniciar sem token**, a menos que você passe `-allow-no-auth`.
- **Token**: defina `CARDMAKER_API_TOKEN`. Todas as rotas, exceto `/healthz`, exigem
  `Authorization: Bearer <token>`. O token vem de variável de ambiente (não de flag) para não aparecer
  na lista de processos. Sem HTTPS na frente, o token trafega em texto puro: em produção, coloque o
  servidor atrás de um proxy reverso com TLS.
- **Limites**: corpo de até 20 MiB (configurável), arte de até 40 milhões de pixels, escala de 1 a 3, no
  máximo `-max-concurrent` renderizações ao mesmo tempo; o excedente espera `-queue-timeout` e recebe
  `503` com `Retry-After`.
- **Arte**: só PNG, JPEG e WebP são aceitos (SVG é rejeitado, então a arte não executa script nem busca
  recursos externos); o tamanho é conferido pelo cabeçalho antes de decodificar a imagem.
- **Não há limite de taxa por cliente** (só o limite de renderizações simultâneas). Se a API for exposta
  além de um uso interno, coloque na frente um proxy com limite de taxa.
- **CORS não está habilitado**: páginas de outra origem não conseguem chamar a API direto do navegador.
  Chame de um backend, ou coloque um proxy que acrescente os cabeçalhos CORS.
- **Erros internos não vazam detalhes**: o cliente recebe uma mensagem genérica e o detalhe vai para o
  log do servidor.
- O navegador usado na renderização só enxerga o tema e a arte enviada, por um servidor local
  temporário (loopback, porta e token aleatórios).

## Formato das respostas

Rotas JSON respondem com um envelope; rotas de imagem respondem com os bytes da imagem e, em caso de erro,
com o mesmo envelope:

```json
{ "success": false, "data": null, "error": "cards[0].rarity: unknown rarity \"mythic\", theme \"default\" defines: common, uncommon, rare, epic, legendary" }
```

| Status | Quando |
|---|---|
| `200` | Sucesso. |
| `400` | Parâmetro ou dado inválido (a mensagem diz qual campo e como corrigir). |
| `401` | Token ausente ou inválido (`WWW-Authenticate: Bearer`). |
| `404` | Tema desconhecido, ou tema sem verso. |
| `405` | Método não permitido na rota. |
| `413` | Corpo maior que o limite. |
| `500` | Falha interna ao renderizar. |
| `503` | Servidor ocupado; tente de novo (`Retry-After`). |

## Rotas

### `GET /healthz`

Sem autenticação. `{"success":true,"data":{"status":"ok"},"error":null}`.

### `GET /v1/themes`

Lista os temas disponíveis, útil para montar seletores de raridade, elemento e variante:

```json
{ "success": true, "error": null, "data": [
  { "name": "default", "size": { "width": 750, "height": 1050 },
    "rarities": ["common", "uncommon", "rare", "epic", "legendary"],
    "elements": ["beast", "fire", "water", "nature", "wind", "light", "dark",
                 "electric", "earth", "ice", "mystic"],
    "variants": ["fullart"], "hasBack": true } ] }
```

### `POST /v1/render`

Renderiza uma carta e responde com a imagem (`image/png` ou `image/webp`).

**Query**

| Parâmetro | Obrigatório | Descrição |
|---|---|---|
| `theme` | sim | Nome do tema (veja `/v1/themes`). |
| `scale` | não | Inteiro de 1 a 3. Padrão `1`. A imagem tem `size × scale` pixels. |
| `format` | não | `png` (padrão) ou `webp`. |
| `quality` | não | Só para `webp`: de 1 a 100, padrão `90` (com perda, visualmente idêntico). `100` é sem perda. |
| `locale` | não | Idioma dos textos. Padrão `pt-BR`. |
| `fallback` | não | Idioma usado quando falta tradução. Padrão: o próprio `locale`. |

**Corpo**: `multipart/form-data` com duas partes:

- `card`: os dados da carta em JSON, **no mesmo formato de uma carta do `cards.json`** (veja
  [CONTRATO.md](CONTRATO.md)). O campo `art` é ignorado: a arte vem da outra parte. A parte pode ser
  enviada como campo de texto ou como arquivo (é o que `fetch` e os navegadores fazem com um `Blob`).
- `art`: o arquivo de arte (PNG, JPEG ou WebP).

A carta passa pelas mesmas validações do `build`: identificadores, raridade, elemento e variante
existentes no tema, e `name` disponível no idioma pedido.

```bash
curl -X POST "http://127.0.0.1:8080/v1/render?theme=default&scale=2" \
  -H "Authorization: Bearer $CARDMAKER_API_TOKEN" \
  -F 'card={"id":"lunaris","number":32,"rarity":"rare","element":"dark","name":{"pt-BR":"Lunaris"},"description":{"pt-BR":"Sua pelagem brilha com a luz da lua."}};type=application/json' \
  -F "art=@lunaris.png" \
  -o lunaris.png
```

Para a versão full art, acrescente `"variant":"fullart"` ao JSON da carta.

### `GET /v1/themes/{name}/back`

Imagem do verso do tema, com os mesmos parâmetros `scale` e `format`. `404` se o tema não define verso.

## Exemplo em JavaScript

Testado com Node 24 contra o servidor real (funciona igual no navegador).

```js
async function renderCard({ theme, card, art, scale = 1, format = "png" }) {
  const form = new FormData();
  form.append("card", new Blob([JSON.stringify(card)], { type: "application/json" }));
  form.append("art", new Blob([art]), "art.png");

  const response = await fetch(
    `${baseUrl}/v1/render?theme=${theme}&scale=${scale}&format=${format}`,
    { method: "POST", body: form, headers: { Authorization: `Bearer ${token}` } },
  );

  if (!response.ok) {
    const { error } = await response.json(); // erros usam o envelope JSON
    throw new Error(`${response.status}: ${error}`);
  }
  return Buffer.from(await response.arrayBuffer()); // no navegador: await response.blob()
}
```

## Desempenho e operação

- Cada carta leva cerca de **1 segundo** com o Chrome (medido em 750×1050, 1x e 2x). Doze requisições
  simultâneas com `-max-concurrent 4` terminam em cerca de 6 s, todas com sucesso.
- O servidor **não faz cache** de imagens: o mesmo pedido é renderizado de novo. Para uma coleção, prefira
  o `build`, que só re-renderiza o que mudou.
- O navegador sobe uma vez com o servidor e é encerrado no desligamento; cada imagem usa uma aba própria.
- Em contêiner, use a imagem do projeto (veja [DOCKER.md](DOCKER.md)): ela já trata o sandbox do Chromium,
  roda sem privilégios e inclui um `docker-compose.yml` com as restrições recomendadas.
