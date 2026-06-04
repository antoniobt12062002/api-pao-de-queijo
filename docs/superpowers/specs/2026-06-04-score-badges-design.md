# Design: Score & Badges (Feature 6)

**Data:** 2026-06-04
**Status:** Aprovado
**Issue:** #7
**Branch:** `feat-score-badges`
**Depende de:** feature/config, feature/rotation, feature/round, feature/participation, feature/notification

---

## Visão Geral

Implementa o módulo de gamificação: score de justiça por usuário e sistema de badges/pins. O score é recalculado automaticamente ao fechar cada rodada. Badges são concedidos com base em critérios permanentes ou mensais.

---

## Domínio

### Score

```
user_id              uuid (PK, FK → users)
times_paid           int       -- rodadas em que foi pagador (status = closed)
times_participated   int       -- rodadas em que participou
total_amount_spent   numeric   -- SUM(todas_participações.quantity * price_per_unit) das rodadas pagas
skip_count           int       -- vezes que cancelou sua vez de pagar
current_streak       int       -- rodadas consecutivas em que participou
score                numeric   -- calculado e persistido pelo ScoreUpdater
updated_at           timestamp
```

`total_amount_spent` = total da conta bancada pelo pagador em cada rodada (soma de todas as participações × price_per_unit), acumulado historicamente.

### Badge

```
id         uuid (PK)
user_id    uuid (FK → users)
type       enum: queijeiro_fiel | papai_noel | nunca_foge | novo_na_fila | big_spender
period     string (nullable) -- "YYYY-MM" para badges mensais; NULL para permanentes
earned_at  timestamp
UNIQUE (user_id, type, period)
```

---

## Fórmula do Score

```
ratio_pago_participado   = times_paid / times_participated   (0 se times_participated = 0)
valor_gasto_normalizado  = total_amount_spent / max(total_amount_spent de todos)  (0 se todos = 0)
taxa_ausencia            = skip_count / (times_paid + skip_count)  (0 se denominador = 0)
streak_bonus             = min(current_streak, 10) / 10

score = max(0,
    (ratio_pago_participado  * 40)
  + (valor_gasto_normalizado * 30)
  - (taxa_ausencia           * 20)
  + (streak_bonus            * 10)
)
```

Score máximo: 100. Mínimo: 0 (clampado).

---

## Critérios dos Badges

| Badge | Tipo | Critério |
|-------|------|---------|
| `novo_na_fila` | permanente | Primeiro pagamento: `times_paid == 1` |
| `nunca_foge` | permanente | `skip_count == 0` e `times_paid > 0` |
| `queijeiro_fiel` | permanente | `current_streak >= 30` |
| `papai_noel` | mensal | Pagador da rodada com mais participantes do mês corrente |
| `big_spender` | mensal | Maior spending acumulado no mês (calculado via query em rounds/participations) |

`papai_noel` e `big_spender` são reavaliados a cada execução do `BadgeChecker`. Idempotência via `INSERT ... ON CONFLICT DO NOTHING` com constraint `UNIQUE (user_id, type, period)`.

---

## Arquitetura

### Componentes

```
internal/domain/score.go          -- Score entity, Badge entity, BadgeType enum,
                                     ScoreRepository interface, BadgeRepository interface,
                                     BadgeChecker interface (expande o atual score.go)
internal/repository/postgres/score.go  -- ScoreRepository (Upsert, GetAll, GetByUserID)
internal/repository/postgres/badge.go  -- BadgeRepository (Upsert, GetByUserID)
internal/score/updater.go         -- ScoreUpdater: atualiza score de pagador + participantes
internal/score/badge_checker.go   -- BadgeChecker: concede badges após score atualizado
internal/usecase/score.go         -- leitura: GetRanking, GetUserScore, GetUserBadges
internal/handler/http/score.go    -- handlers dos 3 endpoints
cmd/main.go                       -- substitui noopScore pela implementação real
```

### Fluxo de Atualização (dentro de ParticipationWindowCloser)

```
round.close()
  → ScoreUpdater.UpdateAfterRound(roundID)
      1. busca round + participações + price_per_unit
      2. atualiza score do pagador (times_paid++, total_amount_spent += conta, recalcula score)
      3. atualiza score de cada participante (times_participated++, current_streak++, recalcula)
      4. reseta current_streak de usuários que NÃO participaram
      5. recalcula score de todos (valor_gasto_normalizado depende do max global)
      6. persiste todos via ScoreRepository.Upsert
  → BadgeChecker.CheckAfterRound(roundID)
      1. verifica badges permanentes para pagador e participantes
      2. reavalia papai_noel e big_spender do mês corrente
      3. persiste via BadgeRepository.Upsert (ON CONFLICT DO NOTHING)
```

### Interfaces

```go
type ScoreRepository interface {
    Upsert(s *Score) error
    GetAll() ([]*Score, error)
    GetByUserID(userID string) (*Score, error)
}

type BadgeRepository interface {
    Upsert(b *Badge) error
    GetByUserID(userID string) ([]*Badge, error)
}

type BadgeChecker interface {
    CheckAfterRound(roundID string) error
}
```

`ScoreUpdater` (já existe em `domain/score.go`) mantém sua assinatura `UpdateAfterRound(roundID string) error`.

---

## Endpoints

| Método | Path | Resposta |
|--------|------|---------|
| `GET` | `/v1/scores` | `[]ScoreResponse` ordenado por `score DESC` |
| `GET` | `/v1/scores/:user_id` | `ScoreResponse` |
| `GET` | `/v1/badges/:user_id` | `[]Badge` |

`ScoreResponse` inclui todos os campos de `Score` mais `user_name` e `user_email` (join com users).

Erros:
- `404` se `user_id` não existir
- `401` se token ausente/inválido

---

## Migrations

- `000008_create_scores_table.up.sql` — tabela `scores`
- `000009_create_badges_table.up.sql` — tabela `badges` com constraint UNIQUE

---

## Testes

- Unit tests para `ScoreUpdater` com mock de repositórios (casos: primeiro pagamento, streak, reset de streak, fórmula)
- Unit tests para `BadgeChecker` (cada badge em isolamento)
- Unit tests para `ScoreUseCase`
- Handler tests para os 3 endpoints
