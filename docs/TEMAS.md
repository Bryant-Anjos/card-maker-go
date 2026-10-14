# Guia de temas

Um tema é uma pasta com um `theme.json` (formato em [CONTRATO.md](CONTRATO.md)) e os arquivos que ele cita:
templates HTML, CSS, fontes, molduras e ícones. O programa não sabe nada de raridades ou elementos: tudo
que aparece na carta é decidido pelo tema.

![Anatomia de uma carta do tema default](images/anatomia.jpg)

*A carta do tema `default` e as peças que o template compõe.*

## Como uma carta vira imagem

1. O template do tema (`template` no `theme.json`, ou o `template` da variante quando a carta tem
   `variant`) é executado com os dados da carta, usando
   [`html/template`](https://pkg.go.dev/html/template) do Go (os textos são escapados automaticamente).
2. A página resultante é aberta em um Chrome/Chromium headless, com viewport do tamanho do tema
   (`size`) e `deviceScaleFactor` igual à escala pedida. Escreva o CSS sempre em pixels do tamanho
   base: a escala 2x sai nítida sem nenhuma mudança no tema.
3. O renderer espera as fontes (`document.fonts.ready`), o trabalho assíncrono do próprio tema
   (`window.cardReady`, veja abaixo) e as imagens carregarem, e tira um screenshot PNG. O **fundo é transparente**: o que o tema não pintar sai com alfa 0 (útil para cantos
   arredondados e brilho).

A página é servida por um servidor local temporário (somente loopback, porta e token aleatórios). Por
isso, **caminhos relativos funcionam normalmente**: `href="style.css"`, `url("fonts/x.woff2")`,
`src="icons/fire.svg"` são resolvidos a partir da raiz do tema.

Nomes reservados na raiz do tema: `__page.html` (a página gerada) e `__art` (a arte da carta). Não use
arquivos com esses nomes.

## Dados do template da carta

| Campo | Tipo | Exemplo de uso |
|---|---|---|
| `.Card.ID` | texto | `class="card-{{.Card.ID}}"` |
| `.Card.Number` | inteiro (0 = sem número) | `{{if .Card.Number}}{{printf "%03d" .Card.Number}}{{end}}` → `032` |
| `.Card.Name` | texto já no locale da saída | `<h1>{{.Card.Name}}</h1>` |
| `.Card.Description` | texto já no locale (pode ser vazio) | `{{if .Card.Description}}…{{end}}` |
| `.Card.Rarity`, `.Card.Element` | ids | |
| `.Card.Variant` | id da variante, vazio no layout padrão | `class="variant-{{.Card.Variant}}"` |
| `.Rarity.ID`, `.Rarity.Color`, `.Rarity.Frame` | a raridade como definida no tema | `style="--c: {{.Rarity.Color}}"` |
| `.Element` | elemento do tema, ou **nulo** se a carta não tem | `{{if .Element}}{{inline .Element.Icon}}{{end}}` |
| `.Size.Width`, `.Size.Height` | tamanho base em px | |
| `.ArtURL` | URL da arte da carta | `<img src="{{.ArtURL}}">` |
| `.Stylesheet` | o `stylesheet` do tema | `<link rel="stylesheet" href="{{.Stylesheet}}">` |

O template do verso (`back`) recebe só `.Size` e `.Stylesheet`.

### Scripts do tema: `window.cardReady`

Um tema pode ter JavaScript, por exemplo para ajustar texto depois que as fontes carregam (as medidas
dependem delas). Se o tema precisa terminar algum trabalho antes da captura, ele atribui uma **promessa** a
`window.cardReady`; o renderer a aguarda. Se a promessa for rejeitada, a renderização falha com o erro; se
nunca terminar, a renderização falha por timeout (60 s por padrão). Temas que não definem `cardReady`
funcionam normalmente.

```html
<script>
  window.cardReady = document.fonts.ready.then(() => { /* mede e ajusta o layout */ });
</script>
```

Arquivos de script do tema são servidos como os demais (`<script src="fit.js">`).

### Funções

- `inline "caminho"`: insere um arquivo do tema (normalmente SVG) direto no HTML, para que ele possa ser
  estilizado pelo CSS da página (por exemplo, usar variáveis CSS de cor). O caminho precisa estar dentro
  do tema.
- Todas as funções padrão do `html/template`: `printf`, `if`, `with`, `eq`, `len`, etc.

## Variantes de layout

Uma variante é um layout alternativo com template próprio, declarado em `variants` no `theme.json`. A
carta escolhe com `"variant": "<id>"`. Raridades e elementos continuam valendo, então uma variante bem
escrita funciona para todas as raridades. Para não duplicar código entre layouts, coloque o que é comum
em arquivos e use `{{inline "arquivo.svg"}}` nos dois templates (o tema `default` faz isso com `frame.svg`).

## Boas práticas

- **Embuta as fontes no tema** (`@font-face` com arquivos locais). Fontes do sistema mudam de máquina
  para máquina; fontes remotas deixam o build lento e não reprodutível.
- **Evite recursos remotos** em geral (imagens, CSS, scripts de CDN) pelo mesmo motivo.
- **Recorte a arte com `object-fit: cover`**: a proporção das ilustrações não é validada.
- **Limite os textos**: nome em uma linha com `text-overflow: ellipsis` e descrição com uma altura máxima
  (`max-height`), para que um texto longo nunca invada outros elementos.
- **Ajuste automático de fonte**: o tema `default` inclui `fit.js`, que encolhe qualquer elemento marcado com
  `data-fit` até o texto caber, de um em um pixel, até `data-fit-min`. Só se o texto ainda não couber no mínimo é
  que ele é cortado com reticências. Elementos que já cabem não são tocados. Dois modos:
  - `data-fit` (o nome): o texto cabe em **uma linha** (não passa da largura da caixa); no mínimo, o CSS corta com
    `text-overflow: ellipsis`.
  - `data-fit="lines"` (a descrição): o texto pode usar **quantas linhas a altura da caixa comportar** (o elemento
    precisa de um `max-height`); texto menor cabe em mais linhas. No mínimo, o `fit.js` define
    `-webkit-line-clamp` com as linhas que cabem, e a última termina em reticências.
- **Deixe margem para efeitos** (brilho, ornamentos que saem da borda) dentro do canvas: o que passar do
  tamanho do tema é cortado.
- Não use `<use href="#id">` esperando que os clones herdem classes CSS: no Chrome isso não é confiável.
  Use atributos SVG diretos (`fill`, `stroke`) dentro de `<defs>` reutilizados.

## O tema `default`

Reproduz o [design de referência](reference/card-design-reference.png). As cinco raridades, do Comum ao
Lendário, com moldura, ornamentos e brilho cada vez mais elaborados:

![As cinco raridades do tema default](images/raridades.jpg)

O verso, ao lado da frente:

![Frente e verso de uma carta](images/frente-e-verso.jpg)

Os onze elementos de exemplo (cada ícone tem a sua cor): Fera, Fogo, Água, Natureza, Vento, Luz, Trevas,
Elétrico, Terra, Gelo e Místico.

![Ícones de elemento do tema default](images/elementos.jpg)

| Item | Como funciona |
|---|---|
| Canvas | 750×1050 (5:7, proporção padrão de cartas colecionáveis). A carta ocupa 722×1022 com 14px de margem para o brilho. |
| Layout | Moldura de 20px; arte 682×644 no topo; barra separadora; painel com nome, ícone, divisória e até 3 linhas de descrição; gema da raridade na base. Coordenadas comentadas no topo de `style.css`. |
| Cor da raridade | Vem de `theme.json` (`color`) como `--c`; tons claros e escuros são derivados com `color-mix()`, então mudar a cor no `theme.json` recolore moldura, gema, painel e brilho. |
| Molduras por raridade | `frames/<raridade>.svg` contém só os **ornamentos** (cantos, entalhes, gema, coroa), mais elaborados a cada nível; a moldura base é comum e fica no template. Os SVGs são gerados por `internal/devtools/framegen` (`make theme-frames`). |
| Brilho | Classes `.rarity-<id>` em `style.css` definem a intensidade (nenhum no Comum, máximo no Lendário). |
| Ícones | `icons/<elemento>.svg`, 48×48, cada um com sua própria cor. Os `id` internos (de gradientes, por exemplo) começam com `icon-<elemento>`, porque o ícone é inserido na mesma página que a moldura; um teste confere isso e o tamanho. |
| Fontes | Lora (nome) e Source Sans 3 (textos), licença OFL, em `fonts/` com as licenças. |
| Verso | `back.html.tmpl`: fundo azul-marinho, moldura dourada e rosa dos ventos. |
| Nome longo | O nome começa em 62 px e encolhe até **28 px** para caber (`data-fit-min` no template). Medido: nomes de até ~36 caracteres cabem inteiros mesmo com o ícone de elemento ao lado (27 caracteres → 37 px, 31 → 32 px, 36 → 28 px); acima disso, reticências. |
| Descrição longa | A descrição começa em 39 px e ocupa até 3 linhas. Se não couber, encolhe até **26 px** (`data-fit-min` no template), quando passa a caber em até 4 linhas na mesma altura; acima disso, a última linha termina em reticências. Descrições que já cabiam em 3 linhas não mudam. |
| Moldura compartilhada | `frame.svg` (faixa metálica e filetes), inserida pelos dois layouts. |
| Variante `fullart` | `fullart.html.tmpl`: a arte ocupa toda a janela da moldura (682×982), o texto fica direto sobre a arte, sem caixa, em branco com contorno preto (`paint-order: stroke fill`, para o contorno crescer para fora sem afinar as letras), e um brilho holográfico na cor da raridade marca a versão especial. Moldura, ornamentos, brilho e gema são os da raridade, então serve para todas. **Arte recomendada: retrato, proporção ≈ 0,69 (ex.: 750×1050 ou 1500×2100 para 2x)**; arte quadrada é ampliada e cortada nas laterais. |

A variante `fullart` serve para qualquer raridade (em cima, o layout padrão; embaixo, o mesmo personagem
em full art):

![Layout padrão e variante full art em todas as raridades](images/fullart.jpg)

O nome encolhe sozinho para caber em uma linha, e só os nomes realmente extremos recebem reticências (à
esquerda, o layout padrão; à direita, o full art):

![Ajuste automático do tamanho do nome](images/nomes-longos.jpg)

Diferença em relação à referência: as cartas da imagem de referência são mais estreitas (≈ 3:5); o tema
usa 5:7, que é o padrão do plano. Mudar a proporção exige ajustar as coordenadas de layout em
`style.css`, no template e no gerador de ornamentos, não só o `size` do `theme.json`.

## Regressão visual (golden images)

`pkg/cardmaker/render/chrome/golden_test.go` renderiza as cartas de exemplo com o tema `default` e compara
com as imagens revisadas em `testdata/golden/default/`:

- **Mesmo navegador que gerou os goldens** (registrado em `testdata/golden/default/RENDERER`): comparação
  estrita. Em máquina ociosa o resultado é idêntico; sob carga pesada, alguns pixels de texto antisserrilhado
  chegaram a variar até 3 níveis (de 255). Por isso um pixel só conta como diferente acima de 8 níveis e a
  diferença média precisa ficar abaixo de 0,02: o ruído passa, mas um dígito trocado, um elemento deslocado ou
  um ajuste sutil de cor ainda falham.
- **Outro navegador ou versão**: só tolerância por média de diferença (o antialiasing de texto muda entre
  versões). Mudanças de layout, cor e assets ainda falham; o teste registra que está em modo tolerante.

Depois de uma mudança visual **intencional**: `make golden`, revise as imagens geradas e faça o commit
junto com a mudança. Os testes com navegador são pulados quando nenhum Chrome/Chromium é encontrado
(veja `CARDMAKER_CHROME` em [CLI.md](CLI.md#variáveis-de-ambiente)).
