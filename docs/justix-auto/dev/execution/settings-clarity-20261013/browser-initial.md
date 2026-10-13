# Initial combined final browser observations

Primary CUA, actual Vite source via in-memory fixture on 5192. Both FE1 and FE2
were complete and source leases released before checks. No live API/DB calls.

- Settings at default 2560px: grouped requisites/contacts, compact successful
  verification result, no duplicate verification fact, explicit empty country.
  Document width 2560. Branch/warehouse, staff/roles and personal-profile
  navigation displayed their existing data/actions correctly.
- Settings at 390px: document width 566, failure. Existing dealer-shell variable
  overrides compact sidebar breakpoint, leaving sidebar 244px and main 146px.
  Local card/header wrapping also needs correction. Bounded repair authorized.
- Reservation at 1440px: labelled grouped controls, styled native dates,
  optional terms and selected count visible; redundant request button absent.
- Shared supplier search and ArrowDown/Enter committed the supplier; VIN search
  selected and added first car, then selected second. Count 2. Request succeeded
  with vehicle-1/vehicle-2 quantity '1' each and empty desiredTerms.
- Future-lot selection qty '3', price '25000.50' succeeded. Displayed price
  25 000.50 USD; fixture recorded amountMinor '2500050', currency USD. Supplying
  advance without a date produced the expected error. Native datetime fill
  through this browser tool did not commit dates; no dated-terms success claim.
- Reservation modal at 390px fits (358px outer width, 356px scroll width), but
  the underlying shell still causes document width 497px. Same shell repair.
- RC3/RC4 incomplete: generic empty guidance remains and zero-selection submit
  is enabled (submit validation itself rejects invalid input). Repair scoped
  to contextual copy and mirrored existing selection/availability checks.

No broader validation loop. Browser use paused during approved repair;
implementation-repair.md and final-validation.md record continuation/evidence.
