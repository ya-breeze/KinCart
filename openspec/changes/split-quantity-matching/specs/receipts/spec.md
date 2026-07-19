## ADDED Requirements

### Requirement: Split-quantity matching

A single planned item with quantity N SHALL be able to absorb multiple receipt line items (up to N), so that buying different variants of a planned item does not create duplicates or force extras.

#### Scenario: Two variants match one planned item
- **GIVEN** a planned item "fruit kefir" with quantity 2
- **AND** a receipt with two lines "kefir strawberry" and "kefir peach"
- **WHEN** both lines are matched to "fruit kefir"
- **THEN** both are linked to that one planned item
- **AND** no duplicate/extra planned item is created for the second variant

#### Scenario: Planned item retained while quantity remains
- **GIVEN** a planned item with quantity 2 and one receipt line already linked to it
- **WHEN** the manager links a second receipt line to the same planned item
- **THEN** the planned item is not deleted before the second link is made
- **AND** the second link succeeds

#### Scenario: Planned item marked bought on first match
- **WHEN** at least one receipt line is matched to a planned item
- **THEN** the planned item is marked bought

#### Scenario: Price aggregated across matched variants
- **GIVEN** two receipt lines matched to one planned item priced 40 and 45
- **THEN** the planned item's recorded spend reflects the sum of the matched lines

#### Scenario: Each variant recorded as its own alias
- **GIVEN** "kefir strawberry" and "kefir peach" are matched to planned "fruit kefir"
- **THEN** an alias is upserted for each variant mapping to "fruit kefir"

#### Scenario: Remaining capacity shown in review
- **GIVEN** a planned item with quantity 2 and one line already matched
- **WHEN** the manager reviews remaining receipt lines
- **THEN** the planned item remains a selectable link target showing it can absorb one more line

#### Scenario: Extra beyond quantity still supported
- **GIVEN** a planned item with quantity 1 already matched by one line
- **WHEN** another receipt line clearly belongs to the same planned name
- **THEN** the manager may still link it (over-consuming) or treat it as an extra, and the review does not break either way
