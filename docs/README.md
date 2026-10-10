# Documentação do Card Maker Go

Esta é a porta de entrada da documentação. Se você está chegando agora, comece pelo [README](https://github.com/Bryant-Anjos/card-maker-go#readme) do
projeto e volte aqui para o que precisar.

## Índice

| Documento | Para quem | O que explica |
|---|---|---|
| [CONTRATO.md](CONTRATO.md) | quem prepara dados ou lê a saída | Formato exato das cartas, do tema e do `manifest.json`, regras de validação e versionamento. |
| [INTEGRACAO.md](INTEGRACAO.md) | quem usa as cartas em outro projeto | Como consumir a saída na web, em apps nativos, em engines e em Go. |
| [TEMAS.md](TEMAS.md) | quem cria ou altera o visual | Como funciona um tema, os dados do template, variantes de layout, boas práticas e o tema `default`. |
| [CLI.md](CLI.md) | quem usa os executáveis | Todas as flags, variáveis de ambiente e códigos de saída. |
| [API.md](API.md) | quem integra o servidor HTTP | Rotas, parâmetros, segurança, limites e exemplos. |
| [DOCKER.md](DOCKER.md) | quem roda em contêiner | Imagem, compose, uso do CLI no Docker e o sandbox do Chromium. |
| [ARQUITETURA.md](ARQUITETURA.md) | quem mexe no código | Mapa de pacotes, fluxos, decisões de projeto e como estender. |
| [DESENVOLVIMENTO.md](DESENVOLVIMENTO.md) | quem contribui | Ambiente, `make`, testes, convenções, commits e a política de documentação. |
| [SOLUCAO-DE-PROBLEMAS.md](SOLUCAO-DE-PROBLEMAS.md) | todos | Sintomas e mensagens de erro reais, com causa e solução. |
| [PLAN.md](PLAN.md) | quem quer o histórico | Plano original, decisões iniciais e o registro do progresso de cada entrega. |

Imagens: as de `docs/images/` são geradas pela ferramenta, não desenhadas à mão; há também o
[design de referência](reference/card-design-reference.png) que o tema `default` reproduz.

## Como esta documentação é mantida

A documentação acompanha o código: **quem muda algo documentado atualiza o texto na mesma mudança**. O mapa
completo de "mudou isto, atualize aquilo", os testes que verificam a documentação (`make docs-check`) e o
processo das imagens (`make docs-images`) estão em
[DESENVOLVIMENTO.md](DESENVOLVIMENTO.md#documentação-viva).

O [PLAN.md](PLAN.md) é o **registro histórico** do planejamento e do progresso. Ele não é a fonte para saber
como o programa funciona hoje: para isso, use os documentos acima.
