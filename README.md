# Card Maker Go

Gerador de cartas agnóstico: recebe dados, imagens e um tema e gera as imagens finais das cartas
(PNG/WebP) mais um `manifest.json`, para serem usadas por qualquer plataforma.

Plano completo: [docs/PLAN.md](docs/PLAN.md) · Design de referência: [docs/reference/](docs/reference/card-design-reference.png)

## Uso

```bash
# checa dados, tema e artes sem renderizar
cardmaker validate -cards testdata/cards/cards.json -theme testdata/themes/basic

# gera as imagens + manifest.json em out/pt-BR/
cardmaker build -cards testdata/cards/cards.json -theme testdata/themes/basic \
  -out out -locale pt-BR -scales 1,2 -formats png,webp -renderer fake
```

Reexecutar o `build` só renderiza o que mudou (`-force` ignora o cache). O renderer `chrome`
(padrão) chega com a fase 4; por enquanto use `-renderer fake`.

## Desenvolvimento

```bash
make check   # go vet + go test
make build   # gera bin/cardmaker
```

Requer Go (veja `.tool-versions`) e um Chrome/Chromium para a renderização (fase 4).
