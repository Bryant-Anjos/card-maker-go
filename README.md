# Card Maker Go

Gerador de cartas agnóstico: recebe dados, imagens e um tema e gera as imagens finais das cartas
(PNG/WebP) mais um `manifest.json`, para serem usadas por qualquer plataforma.

Plano completo: [docs/PLAN.md](docs/PLAN.md) · Design de referência: [docs/reference/](docs/reference/card-design-reference.png)

## Desenvolvimento

```bash
make check   # go vet + go test
make build   # gera bin/cardmaker
```

Requer Go (veja `.tool-versions`) e um Chrome/Chromium para a renderização (fase 4).
