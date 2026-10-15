# Governance

The project is led by its maintainers, listed in [MAINTAINERS.md](MAINTAINERS.md).

## How decisions are made

- **Everyday changes** are decided in pull requests. A maintainer reviews and merges them when the checks pass and the change fits the project's [principles](docs/explanation/design.md#principles).
- **Design choices** that are hard to undo, or that change a rule, are written as [decision records](docs/contributing/decisions/README.md). Anyone may propose one in a pull request. A maintainer accepts or declines it, with the reasons in the record. An accepted record is not edited; a new record replaces it.
- **What gets built** follows [the roadmap](docs/roadmap.md), where each feature is scored by demand, impact and effort from public evidence. Anyone may propose a change to a score, with evidence.
- **Disagreements** are settled by discussion. If maintainers still disagree, the lead maintainer decides, and says why.

## Rules no decision overrides

These are part of what the project is. Changing one needs a decision record and the agreement of every maintainer:

- the agent's bytes reach the user's terminal unchanged while attached;
- nothing new listens on a host, and there is no relay and no telemetry;
- the tool stores no secrets.

## Becoming a maintainer

A contributor who has made steady, good changes over some months, and reviews others' work kindly, may be invited by the maintainers. Maintainers who step back are listed as emeritus.

## Code of conduct

Everyone in the project's spaces follows the [Code of Conduct](CODE_OF_CONDUCT.md). Maintainers enforce it.
