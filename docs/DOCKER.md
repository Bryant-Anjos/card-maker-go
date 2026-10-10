# Docker

A imagem traz o Chromium, os dois executáveis (`cardmaker` e `cardmaker-server`) e os temas de `themes/`.
Os binários são estáticos (sem cgo) e rodam como usuário sem privilégios.

```bash
docker build -t cardmaker .
```

Verificado com Docker 29: o build leva cerca de 2,5 minutos (a maior parte é baixar o Chromium e os
módulos Go) e a imagem final tem cerca de 790 MB, quase tudo o Chromium.

## API

O servidor escuta em todas as interfaces dentro do contêiner, então **ele se recusa a iniciar sem token**:

```bash
docker run -d --init -p 127.0.0.1:8080:8080 \
  -e CARDMAKER_API_TOKEN=um-token-longo-e-aleatorio cardmaker
```

Ou, com todas as restrições de segurança recomendadas, pelo `docker-compose.yml` do projeto:

```bash
CARDMAKER_API_TOKEN=um-token-longo-e-aleatorio docker compose up --build
```

O compose publica a porta só em `127.0.0.1`, usa `init: true`, sistema de arquivos somente leitura (com
`tmpfs` em `/tmp`), `cap_drop: ALL`, `no-new-privileges` e limites de memória e de processos. Veja o
[guia da API](API.md) para as rotas e o `/healthz` usado no `HEALTHCHECK`. Para expor além de
`localhost`, coloque na frente um proxy reverso com TLS.

## CLI

Para gerar uma coleção sem instalar Go nem o navegador, monte os dados (somente leitura) e a saída, e use
o seu próprio usuário para os arquivos gerados não ficarem como `root`:

```bash
docker run --rm --user "$(id -u):$(id -g)" \
  -v "$PWD/dados:/in:ro" -v "$PWD/out:/out" \
  cardmaker cardmaker build -cards /in/cards.json -theme /app/themes/default \
  -out /out -scales 1,2 -formats png,webp
```

O tema `default` já está em `/app/themes/default`; para um tema seu, monte a pasta e aponte `-theme` para ela.

## O sandbox do Chromium

Dentro de contêineres, o Chromium costuma não conseguir criar o sandbox (que depende de *user namespaces*
bloqueados pelo perfil padrão do Docker). Testado: com o sandbox ligado, o navegador não sobe
(`No usable sandbox!`). Por isso a imagem define `CARDMAKER_CHROME_NO_SANDBOX=1`, que acrescenta
`--no-sandbox` ao navegador. O mesmo controle existe fora do Docker: flag `-no-sandbox` nos dois
executáveis ou a variável `CARDMAKER_CHROME_NO_SANDBOX`.

**O que isso significa:** sem o sandbox, uma falha no navegador ao decodificar uma arte maliciosa não
estaria contida pelo próprio navegador, e sim só pelo contêiner. Por isso a imagem roda como usuário sem
privilégios (UID 10001) e o compose remove todas as capabilities, impede escalonamento de privilégios e
deixa o sistema de arquivos somente leitura. A API só aceita PNG, JPEG e WebP como arte (nunca SVG) e só
renderiza cartas com o tema carregado no servidor. Se preferir manter o sandbox, use um perfil
`seccomp` que permita *user namespaces* (`--security-opt seccomp=perfil.json`) e defina
`CARDMAKER_CHROME_NO_SANDBOX=0`.

## Detalhes úteis

- **`--init`**: o Chromium cria processos filhos que precisam ser recolhidos; use `--init` (o compose já faz).
- **`HOME=/tmp`**: dá ao navegador um diretório de perfil gravável com qualquer UID e com o sistema de
  arquivos somente leitura.
- **Comparação com o host**: a mesma carta renderizada no contêiner (Chromium do Debian) e no host
  (Chrome 151) diferiu em média 0,04 de 255 por canal, ou seja, praticamente idêntica. As imagens de
  referência dos testes (`testdata/golden`) são exatas apenas com o navegador que as gerou.
